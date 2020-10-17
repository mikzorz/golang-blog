package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Integration test
func TestWebIntegration(t *testing.T) {
	t.Run("/all returns all articles", func(t *testing.T) {
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		articles := MakeBothTypesOfArticle(20)

		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/all")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)

		for _, v := range articles {
			assertContains(t, resp.Body.String(), v.Title)
		}
	})

	t.Run("get pages of articles", func(t *testing.T) {
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		progWant, otherWant := MakeSeparatedArticles(50)

		// Does "Last Edited:" show on index page?
		progWant[len(progWant)-1].Edited = myTimeToString(time.Now().UTC().Add(time.Hour * 1).Add(time.Second * 50))

		articles := append(progWant, otherWant...)

		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		progWant, otherWant = reverseArticles(progWant), reverseArticles(otherWant)

		cases := []struct {
			path string
			want []Article
		}{
			{"/", progWant[:perPage]},
			{"/page/1", progWant[:perPage]},
			{"/page/2", progWant[perPage : perPage*2]},
			{"/page/-5", progWant[:perPage]},
			{"/page/9999", progWant[len(progWant)-perPage : len(progWant)]},
			{"/page/abc", progWant[:perPage]},
			{"/other", otherWant[:perPage]},
			{"/other/page/1", otherWant[:perPage]},
			{"/other/page/2", otherWant[perPage : perPage*2]},
			{"/other/page/-5", otherWant[:perPage]},
			{"/other/page/9999", otherWant[len(otherWant)-perPage : len(otherWant)]},
			{"/other/page/abc", otherWant[:perPage]},
		}

		for _, c := range cases {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, c.path)

			server.ServeHTTP(resp, req)

			for _, a := range c.want {
				assertContains(t, resp.Body.String(), a.Title)
			}
		}

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/")

		server.ServeHTTP(resp, req)

		assertContains(t, resp.Body.String(), "Published: "+progWant[0].Published[:10])
		assertContains(t, resp.Body.String(), "Last Edited: "+progWant[0].Edited[:10])
	})

	t.Run("index with only one page", func(t *testing.T) {
		// No asserts, just see if it works. Caused errors because of out of bounds pagination.
		articles := MakeArticlesOfCategory(perPage-1, time.Now(), progCat)
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/")
		server.ServeHTTP(resp, req)
	})

	t.Run("get view page of single article", func(t *testing.T) {
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		articles := MakeArticlesOfCategory(10, time.Now().UTC(), progCat)

		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		// Valid article
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/"+articles[0].Slug)

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertContains(t, resp.Body.String(), articles[0].Title)
		assertContains(t, resp.Body.String(), articles[0].Body)

		// Non-existent article
		resp = httptest.NewRecorder()
		req = newGetRequest(t, "/does-not-exist")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 404)
	})

	t.Run("new article submission", func(t *testing.T) {
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		store, closeDB := NewFileSystemStore(tmpFile)
		defer closeDB()
		server := NewServer(store)

		t.Run("get new article form page", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/new", nil)
			server.ServeHTTP(resp, req)

			// does resp contain form? Can I just check if status code is 200?
			assertStatus(t, resp.Code, 200)
			assertContains(t, resp.Body.String(), "<form class=\"\" action=\"/\" method=\"post\">")
			assertContains(t, resp.Body.String(), "New Article") // May change this to be more specific.(?)
		})

		t.Run("form page should show validation errors after submitting invalid article", func(t *testing.T) {
			invalidArticle := Article{
				Title:    strings.Repeat("this is an invalid title ", 100),
				Preview:  "",
				Body:     "",
				Slug:     "&$+,/:;=?@# <>[]{}|\\^%",
				Category: "invalid-category",
			}
			data := url.Values{}
			data.Set("title", invalidArticle.Title)
			data.Set("preview", invalidArticle.Preview)
			data.Set("body", invalidArticle.Body)
			data.Set("slug", invalidArticle.Slug)
			data.Set("category", invalidArticle.Category)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)
			// Are validation errors returned and displayed?
			assertContains(t, resp.Body.String(), errTitleLong)
			assertContains(t, resp.Body.String(), errPreviewEmpty)
			assertContains(t, resp.Body.String(), errBodyEmpty)
			assertContains(t, resp.Body.String(), errSlugBad)
			assertContains(t, resp.Body.String(), errCatInvalid)

			// Does the input remain in the input fields after validation instead of wiping it all clean?
			newA := invalidArticle
			newA.Preview = "Valid Preview"
			newA.Body = "Valid Body"

			data = url.Values{}
			data.Set("title", newA.Title)
			data.Set("preview", newA.Preview)
			data.Set("body", newA.Body)
			data.Set("slug", newA.Slug)
			data.Set("category", newA.Category)

			resp = httptest.NewRecorder()
			req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)

			assertContains(t, resp.Body.String(), newA.Title)
			assertContains(t, resp.Body.String(), newA.Preview)
			assertContains(t, resp.Body.String(), newA.Body)
			assertContains(t, resp.Body.String(), newA.Slug)
			assertContains(t, resp.Body.String(), newA.Category)
		})

		t.Run("should be able to submit and save new valid article", func(t *testing.T) {
			validArticle := validArticleBase
			data := url.Values{}
			data.Set("title", validArticle.Title)
			data.Set("preview", validArticle.Preview)
			data.Set("body", validArticle.Body)
			data.Set("slug", validArticle.Slug)
			data.Set("category", validArticle.Category)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, http.StatusAccepted)
			_, saved := store.getArticle(validArticle.Slug)
			if saved != (Article{}) {
				assertArticleWithoutTime(t, saved, validArticle)
			} else {
				t.Error("valid article could not be found in store")
			}

			// Will likely be changed to redirect to admin panel instead of index.
			// Can I check the url?
			assertContains(t, resp.Body.String(), "Index")
			assertContains(t, resp.Body.String(), validArticle.Title)
			// t.Error("not done, test redirect")
		})

		t.Run("article with slug that already exists returns error", func(t *testing.T) {
			validArticle := validArticleBase
			validArticle.Slug = "This-should-not-be-saved-twice"
			validArticle.Published = myTimeToString(time.Now().UTC())
			validArticle.Edited = myTimeToString(time.Now().UTC())

			numOfArts := len(store.getAll())

			// Save first article
			store.newArticle(validArticle)
			if len(store.getAll()) != (numOfArts + 1) {
				t.Fatal("failed to save valid article, not finishing this test")
			}

			// Send second article
			data := setDataValues(validArticle)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			if len(store.getAll()) != (numOfArts + 1) {
				t.Error("saved an article with a slug that is already in use")
			}

			// Send third article, same slug but case shifted. Should not be saved.
			data = url.Values{}
			data.Set("title", validArticle.Title)
			data.Set("preview", validArticle.Preview)
			data.Set("body", validArticle.Body)
			data.Set("slug", "tHiS-sHoUlD-nOt-Be-SaVeD-tWiCe")
			data.Set("category", validArticle.Category)

			resp = httptest.NewRecorder()
			req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			if len(store.getAll()) != (numOfArts + 1) {
				t.Error("saved an article with a slug that is already in use but with a different case")
			}
		})
	})

	t.Run("edit article", func(t *testing.T) {
		a := validArticleBase
		a.Slug = "valid-article"
		a.Published = myTimeToString(time.Now().UTC().Add(time.Hour * time.Duration(-1)))
		a.Edited = myTimeToString(time.Now().UTC())

		t.Run("edit article form", func(t *testing.T) {

			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()
			store, closeDB := NewFileSystemStore(tmpFile, []Article{a})
			defer closeDB()
			server := NewServer(store)

			t.Run("article exists", func(t *testing.T) {
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/"+a.Slug+"/edit", nil)
				server.ServeHTTP(resp, req)

				assertStatus(t, resp.Code, 200)
				// Check that there is a form with each input field filled with article's data.
				assertContains(t, resp.Body.String(), "<input type=\"text\" name=\"title\" value=\""+a.Title+"\">")
				assertContains(t, resp.Body.String(), template.HTMLEscapeString(a.Preview)+"</textarea>")
				assertContains(t, resp.Body.String(), template.HTMLEscapeString(a.Body)+"</textarea>")
				assertContains(t, resp.Body.String(), "<input type=\"text\" name=\"slug\" value=\""+a.Slug+"\">")
				assertContains(t, resp.Body.String(), "<input type=\"text\" name=\"category\" value=\""+a.Category+"\">")
			})
			t.Run("article does not exist", func(t *testing.T) {
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/does-not-exist/edit", nil)
				server.ServeHTTP(resp, req)

				assertStatus(t, resp.Code, 404)
			})
		})

		t.Run("successfully edit existing article", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()
			store, closeDB := NewFileSystemStore(tmpFile, []Article{a})
			defer closeDB()
			server := NewServer(store)

			edit := editedBase
			data := setDataValues(edit)
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/"+a.Slug, strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			// after successful edit, render view of article. TEMP
			assertContains(t, resp.Body.String(), "<article class=\"content\">")
			assertContains(t, resp.Body.String(), edit.Title)
			assertContains(t, resp.Body.String(), "Published: "+a.Published[:10])
			assertContains(t, resp.Body.String(), "Last Edited: "+myTimeToString(time.Now().UTC())[:10])
			assertContains(t, resp.Body.String(), edit.Body)
			assertNotContain(t, resp.Body.String(), errSlugAlreadyExists)
		})

		t.Run("failing to edit article shows validation errors and retains edited fields", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()
			store, closeDB := NewFileSystemStore(tmpFile, []Article{a})
			defer closeDB()
			server := NewServer(store)

			invalidArticle := Article{
				Title:    strings.Repeat("this is an invalid title ", 100),
				Preview:  "",
				Body:     "",
				Slug:     "&$+,/:;=?@# <>[]{}|\\^%",
				Category: "invalid-category",
			}
			data := url.Values{}
			data.Set("title", invalidArticle.Title)
			data.Set("preview", invalidArticle.Preview)
			data.Set("body", invalidArticle.Body)
			data.Set("slug", invalidArticle.Slug)
			data.Set("category", invalidArticle.Category)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/"+a.Slug, strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)
			// Are validation errors returned and displayed?
			assertContains(t, resp.Body.String(), errTitleLong)
			assertContains(t, resp.Body.String(), errPreviewEmpty)
			assertContains(t, resp.Body.String(), errBodyEmpty)
			assertContains(t, resp.Body.String(), errSlugBad)
			assertContains(t, resp.Body.String(), errCatInvalid)

			// Does the input remain in the input fields after validation instead of wiping it all clean?
			newA := invalidArticle
			newA.Preview = "Valid Preview"
			newA.Body = "Valid Body"

			data = url.Values{}
			data.Set("title", newA.Title)
			data.Set("preview", newA.Preview)
			data.Set("body", newA.Body)
			data.Set("slug", newA.Slug)
			data.Set("category", newA.Category)

			resp = httptest.NewRecorder()
			req, _ = http.NewRequest(http.MethodPost, "/"+a.Slug, strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)

			assertContains(t, resp.Body.String(), newA.Title)
			assertContains(t, resp.Body.String(), newA.Preview)
			assertContains(t, resp.Body.String(), newA.Body)
			assertContains(t, resp.Body.String(), newA.Slug)
			assertContains(t, resp.Body.String(), newA.Category)
		})
	})

	t.Run("delete article", func(t *testing.T) {
		articles := MakeArticlesOfCategory(10, time.Now(), progCat)
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		t.Run("fail to delete non-existing article, receive code 404", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/does-not-exist", nil)
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 404)

			if len(store.getAll()) != len(articles) {
				t.Error("article should not be deleted")
			}
		})

		t.Run("successfully delete existing article, receive code 202", func(t *testing.T) {
			toDelete := articles[2]
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/"+toDelete.Slug, nil)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 202)

			if len(store.getAll()) != len(articles)-1 {
				t.Error("article not deleted")
			}

			// Go to index to check if article is gone.
			resp = httptest.NewRecorder()
			req = newGetRequest(t, "/")
			server.ServeHTTP(resp, req)

			assertNotContain(t, resp.Body.String(), toDelete.Title)
		})
	})
}

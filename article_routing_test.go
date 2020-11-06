package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestArticleRouting(t *testing.T) {
	t.Run("get all, routing", func(t *testing.T) {
		store := StubStore{calls: []string{}}
		sessStore := StubSessionStore{}
		server := NewServer(&store, &sessStore)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/all")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertCalls(t, store.calls, []string{"getAll"})
	})

	t.Run("get pages of articles, routing", func(t *testing.T) {
		store := StubStore{calls: []string{}}
		sessStore := StubSessionStore{}
		server := NewServer(&store, &sessStore)

		cases := []struct {
			path string
		}{
			{"/"},
			{"/page/1"},
			{"/page/2"},
			{"/page/-5"},
			{"/page/9999"},
			{"/page/abc"},
			{"/other"},
			{"/other/page/1"},
			{"/other/page/2"},
			{"/other/page/-5"},
			{"/other/page/9999"},
			{"/other/page/abc"},
		}

		want := []string{}

		for _, c := range cases {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, c.path)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 200)
			want = append(want, "getPage")
		}

		assertCalls(t, store.calls, want)
	})

	t.Run("get view page of single article", func(t *testing.T) {
		articles := MakeArticlesOfCategory(10, time.Now().UTC(), progCat)

		store := StubStore{articles: articles, calls: []string{}}
		sessStore := StubSessionStore{}
		server := NewServer(&store, &sessStore)

		// Valid article
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/programming-article-1")

		server.ServeHTTP(resp, req)

		want := []string{"getArticle"}

		assertStatus(t, resp.Code, 200)
		assertCalls(t, store.calls, want)

		// Non-existent article
		resp = httptest.NewRecorder()
		req = newGetRequest(t, "/does-not-exist")

		server.ServeHTTP(resp, req)

		want = append(want, "getArticle")

		assertStatus(t, resp.Code, 404)
		assertCalls(t, store.calls, want)
	})

	t.Run("article should show date but not time", func(t *testing.T) {
		article := MakeArticleOfCategory(1, time.Now().UTC(), progCat)

		store := StubStore{articles: []Article{article}}
		sessStore := StubSessionStore{}
		server := NewServer(&store, &sessStore)

		t.Run("main index page", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, "/")

			server.ServeHTTP(resp, req)

			want := article.Published[:10]
			notWant := article.Published[11:]

			assertContains(t, resp.Body.String(), want)
			assertNotContain(t, resp.Body.String(), notWant)

			resp = httptest.NewRecorder()
			req = newGetRequest(t, "/all")

			server.ServeHTTP(resp, req)

			assertContains(t, resp.Body.String(), want)
			assertNotContain(t, resp.Body.String(), notWant)
		})

		t.Run("other index page", func(t *testing.T) {
			article := MakeArticleOfCategory(1, time.Now().UTC(), otherCat)

			store := StubStore{articles: []Article{article}}
			sessStore := StubSessionStore{}
			server := NewServer(&store, &sessStore)

			resp := httptest.NewRecorder()
			req := newGetRequest(t, "/other")

			server.ServeHTTP(resp, req)

			want := article.Published[:10]
			notWant := article.Published[11:]

			assertContains(t, resp.Body.String(), want)
			assertNotContain(t, resp.Body.String(), notWant)
		})

		t.Run("article page", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, "/programming-article-1")

			server.ServeHTTP(resp, req)

			want := article.Published[:10]
			notWant := article.Published[11:]

			assertContains(t, resp.Body.String(), want)
			assertNotContain(t, resp.Body.String(), notWant)
		})
	})

	t.Run("POST new valid article to /", func(t *testing.T) {
		store := StubStore{articles: []Article{}}
		sessStore := StubSessionStore{Sesh{}}
		server := NewServer(&store, &sessStore)

		want := validArticleBase

		data := setDataValues(validArticleBase)

		t.Run("401 on POST /new if not logged in", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newPostRequest(t, "/new", data)
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 401)
		})

		sessStore.sesh.Authenticated = true
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/new", strings.NewReader(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 303)
		assertCalls(t, store.calls, []string{"new"})
		if len(store.articles) != 0 {
			assertArticleWithoutTime(t, store.articles[0], want)
		} else {
			t.Error("failed to save new valid article")
		}
	})

	t.Run("fail to POST new invalid article to /", func(t *testing.T) {
		store := StubStore{articles: []Article{}, calls: []string{}}
		sessStore := StubSessionStore{Sesh{Authenticated: true}}
		server := NewServer(&store, &sessStore)

		invalidArticles := []Article{}
		for i := 0; i < 5; i++ {
			invalidArticles = append(invalidArticles, validArticleBase)
		}

		invalidArticles[0].Title = strings.Repeat("this is an invalid title ", 100) // too long
		invalidArticles[1].Preview = ""                                             // empty
		invalidArticles[2].Body = ""                                                // empty
		invalidArticles[3].Slug = "&$+,/:;=?@# <>[]{}|\\^%"                         // unsafe or reserved characters
		invalidArticles[4].Category = "invalid-category"                            // not one of the valid categories

		for i := 0; i < len(invalidArticles); i++ {
			notWant := invalidArticles[i]
			data := url.Values{}
			data.Set("title", notWant.Title)
			data.Set("preview", notWant.Preview)
			data.Set("body", notWant.Body)
			data.Set("slug", notWant.Slug)
			data.Set("category", notWant.Category)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/new", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)
			assertCalls(t, store.calls, []string{})
			if len(store.articles) != 0 {
				t.Errorf("saved an invalid article, %v", notWant)
			}
			store.articles = []Article{}
			store.calls = []string{}
		}
	})

	t.Run("fail to POST new empty article to /", func(t *testing.T) {
		store := StubStore{articles: []Article{}, calls: []string{}}
		sessStore := StubSessionStore{Sesh{Authenticated: true}}
		server := NewServer(&store, &sessStore)

		invalidArticles := []Article{}
		for i := 0; i < 5; i++ {
			invalidArticles = append(invalidArticles, validArticleBase)
		}

		invalidArticles[0].Title = ""
		invalidArticles[1].Preview = ""
		invalidArticles[2].Body = ""
		invalidArticles[3].Slug = ""
		invalidArticles[4].Category = ""

		for i := 0; i < len(invalidArticles); i++ {
			empty := invalidArticles[i]
			data := setDataValues(empty)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/new", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 400)
			assertCalls(t, store.calls, []string{})
			if len(store.articles) != 0 {
				t.Errorf("saved an empty article, %v", invalidArticles[i])
			}
			store.articles = []Article{}
			store.calls = []string{}
		}
	})

	t.Run("send POST request to /{slug}", func(t *testing.T) {
		article := validArticleBase
		article.Slug = "some-article"
		article.Published = myTimeToString(time.Now().UTC())
		article.Edited = myTimeToString(time.Now().UTC())

		store := StubStore{articles: []Article{article}, calls: []string{}}
		sessStore := StubSessionStore{Sesh{Authenticated: true}}
		server := NewServer(&store, &sessStore)

		t.Run("303 when editing existing article with new valid values", func(t *testing.T) {
			editedWant := editedBase
			data := setDataValues(editedWant)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/some-article/edit", strings.NewReader(data.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 303)
			assertCalls(t, store.calls, []string{"getArticle", "edit"})
			store.calls = []string{}
		})

		t.Run("404 when trying to edit inexistent article", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/does-not-exist/edit", nil)
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, http.StatusNotFound)
			assertCalls(t, store.calls, []string{"getArticle"})
		})
	})

	t.Run("send DELETE req to {slug}", func(t *testing.T) {
		exists := newValidArticleWithTime()
		exists.Slug = "some-article"

		store := StubStore{articles: []Article{exists}}
		sessStore := StubSessionStore{Sesh{Authenticated: true}}
		server := NewServer(&store, &sessStore)

		t.Run("200 for existing article", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/some-article", nil)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 200)
			assertCalls(t, store.calls, []string{"getArticle", "delete"})
			store.calls = []string{}
		})

		t.Run("404 for existing article", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/does-not-exist", nil)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 404)
			assertCalls(t, store.calls, []string{"getArticle"})
		})
	})
}

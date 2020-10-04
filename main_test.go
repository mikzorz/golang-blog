package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestArticle(t *testing.T) {
	t.Run("get all, routing", func(t *testing.T) {
		store := StubStore{calls: []string{}}
		server := NewServer(&store)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/all")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertCalls(t, store.calls, []string{"getAll"})
	})

	t.Run("get pages of articles, routing", func(t *testing.T) {
		store := StubStore{calls: []string{}}
		server := NewServer(&store)

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

	t.Run("get view page of single article, routing", func(t *testing.T) {
		articles := MakeArticlesOfCategory(10, time.Now(), progCat)

		store := StubStore{articles: articles, calls: []string{}}
		server := NewServer(&store)

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
		article := MakeArticleOfCategory(1, time.Now(), progCat)

		store := StubStore{articles: []Article{article}}
		server := NewServer(&store)

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
			article := MakeArticleOfCategory(1, time.Now(), otherCat)

			store := StubStore{articles: []Article{article}}
			server := NewServer(&store)

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

	validArticle := Article{
		Title:    "POST me!",
		Preview:  "<p>This is a valid preview.</p>",
		Body:     "<p>This is a valid body of an article.</p>",
		Slug:     "this-IS_a.v4l1d~slug",
		Category: "Other",
	}

	t.Run("POST new valid article to /", func(t *testing.T) {
		store := StubStore{articles: []Article{}}
		server := NewServer(&store)

		want := validArticle

		data := url.Values{}
		data.Set("title", want.Title)
		data.Set("preview", want.Preview)
		data.Set("body", want.Body)
		data.Set("slug", want.Slug)
		data.Set("category", want.Category)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertCalls(t, store.calls, []string{"new"})
		if len(store.articles) != 0 {
			assertArticleWithoutTime(t, store.articles[0], want)
		} else {
			t.Error("failed to save new valid article")
		}
	})

	t.Run("fail to POST new invalid article to /", func(t *testing.T) {
		store := StubStore{articles: []Article{}, calls: []string{}}
		server := NewServer(&store)

		invalidArticles := []Article{}
		for i := 0; i < 5; i++ {
			invalidArticles = append(invalidArticles, validArticle)
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
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
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
		server := NewServer(&store)

		invalidArticles := []Article{}
		for i := 0; i < 5; i++ {
			invalidArticles = append(invalidArticles, validArticle)
		}

		invalidArticles[0].Title = ""
		invalidArticles[1].Preview = ""
		invalidArticles[2].Body = ""
		invalidArticles[3].Slug = ""
		invalidArticles[4].Category = ""

		for i := 0; i < len(invalidArticles); i++ {
			empty := invalidArticles[i]
			data := url.Values{}
			data.Set("title", empty.Title)
			data.Set("preview", empty.Preview)
			data.Set("body", empty.Body)
			data.Set("slug", empty.Slug)
			data.Set("category", empty.Category)

			resp := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(data.Encode()))
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
}

// Store test
func TestFileSystemStore(t *testing.T) {
	t.Run("load store", func(t *testing.T) {
		t.Run("saves and doesn't overwrite when calling NewFileSystemStore", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			articles := MakeBothTypesOfArticle(25)
			_, closeDB := NewFileSystemStore(tmpFile, articles)
			closeDB()

			// Normal load, check if 50 articles exist
			store, closeDB := NewFileSystemStore(tmpFile)

			got := store.getAll()

			if len(got) != 50 {
				t.Error("Articles not saved between DB reloads.")
			}

			closeDB()

			// Because tmpFile is not empty, ignore articles, check if 50 articles exist
			articles = MakeBothTypesOfArticle(25)
			store, closeDB = NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got = store.getAll()

			if len(got) != 50 {
				t.Error("Articles were added to a non-empty db file during db reload")
			}
		})
		t.Run("can load dbfile with tables but no articles", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			_, closeDB := NewFileSystemStore(tmpFile)
			closeDB()
			// Blank db setup finished.

			articles := MakeBothTypesOfArticle(25)
			store, closeDB := NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got := store.getAll()

			if len(got) != 50 {
				t.Error("failed to write to an already setup empty db.")
			}
		})
	})

	t.Run("new store", func(t *testing.T) {
		t.Run("get all", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			articles := MakeBothTypesOfArticle(50)
			store, closeDB := NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got := store.getAll()
			want := reverseArticles(articles)
			assertArticles(t, got, want)
		})

		t.Run("get page", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			progWant, otherWant := MakeSeparatedArticles(50)
			store, closeDB := NewFileSystemStore(tmpFile, append(progWant, otherWant...))
			defer closeDB()

			progWant = reverseArticles(progWant)
			otherWant = reverseArticles(otherWant)

			got, p, mxP := store.getPage(1, progCat)

			assertInt(t, p, 1)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, progWant[0:perPage])

			got, p, mxP = store.getPage(-1, progCat)

			assertInt(t, p, 1)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, progWant[0:perPage])

			got, p, mxP = store.getPage(3, otherCat)

			assertInt(t, p, 3)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, otherWant[2*perPage:3*perPage])

			got, p, mxP = store.getPage(6, otherCat)

			assertInt(t, p, 5)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, otherWant[4*perPage:5*perPage])
		})

		t.Run("get single article", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			prog, other := MakeSeparatedArticles(20)
			store, closeDB := NewFileSystemStore(tmpFile, append(prog, other...))
			defer closeDB()

			got := store.getArticle(prog[0].Slug)
			assertArticle(t, got, prog[0])

			got = store.getArticle("does-not-exist")
			assertArticle(t, got, Article{})
		})
	})
}

// Integration test
func TestIntegration(t *testing.T) {
	t.Run("get all", func(t *testing.T) {
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
	})

	t.Run("get view page of single article", func(t *testing.T) {
		tmpFile, cleanTempFile := makeTempFile()
		defer cleanTempFile()
		articles := MakeArticlesOfCategory(10, time.Now(), progCat)

		store, closeDB := NewFileSystemStore(tmpFile, articles)
		defer closeDB()
		server := NewServer(store)

		// Valid article
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/programming-article-1")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertContains(t, resp.Body.String(), "Programming Article 1")

		// Non-existent article
		resp = httptest.NewRecorder()
		req = newGetRequest(t, "/does-not-exist")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 404)

	})
}

type StubStore struct {
	articles []Article
	calls    []string
}

func (s *StubStore) getAll() []Article {
	s.calls = append(s.calls, "getAll")
	return s.articles
}

func (s *StubStore) getPage(page int, category string) ([]Article, int, int) {
	s.calls = append(s.calls, "getPage")

	// Should paginate, but currently don't need to. Pagination is not tested for StubStore.
	return s.articles, 0, 0
}

func (s *StubStore) getArticle(slug string) Article {
	s.calls = append(s.calls, "getArticle")
	for _, a := range s.articles {
		if a.Slug == slug {
			return a
		}
	}
	return Article{}
}

func (s *StubStore) newArticle(a Article) {
	s.calls = append(s.calls, "new")
	s.articles = append(s.articles, a)
}

func newGetRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Errorf("Failed to make new GET request, %s", err)
	}
	return req
}

func MakeSeparatedArticles(n int) (progWant []Article, otherWant []Article) {
	for i := 1; i <= n; i++ {
		progWant = append(progWant, MakeArticleOfCategory(i, time.Now(), progCat))
	}
	for i := 1; i <= n; i++ {
		otherWant = append(otherWant, MakeArticleOfCategory(i, time.Now(), otherCat))
	}
	return
}

func assertInt(t *testing.T, got, want int) {
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func assertArticle(t *testing.T, got, want Article) {
	t.Helper()
	if got != want {
		t.Errorf("articles don't match, got %v, want %v", got, want)
	}
}

func assertArticleWithoutTime(t *testing.T, got, want Article) {
	t.Helper()
	timelessGot := Article{
		Title:    got.Title,
		Preview:  got.Preview,
		Body:     got.Body,
		Slug:     got.Slug,
		Category: got.Category,
	}
	if timelessGot != want {
		t.Errorf("articles don't match, got %v, want %v", timelessGot, want)
	}
}

func assertNotArticle(t *testing.T, got, notwant Article) {
	t.Helper()
	if got == notwant {
		t.Errorf("articles shouldn't match, got %v, don't want %v", got, notwant)
	}
}

func assertArticles(t *testing.T, got, want []Article) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		// Very ugly logs.
		t.Errorf("article slice doesn't match, got %v, want %v", got, want)
	}
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status codes don't match, got %d, want %d", got, want)
	}
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("server calls don't match, got %v, want %v", got, want)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		// t.Errorf("%s does not contain %s", got, want)
		t.Errorf("got %s, want %s", got, want)
	}
}

func assertNotContain(t *testing.T, got, notWant string) {
	t.Helper()
	if strings.Contains(got, notWant) {
		// t.Errorf("%s does not contain %s", got, want)
		t.Errorf("got %s, don't want %s", got, notWant)
	}
}

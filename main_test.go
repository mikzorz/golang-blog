package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestArticle(t *testing.T) {
	t.Run("get all", func(t *testing.T) {
		articles := MakeBothTypesOfArticle(20)

		store := StubStore{articles: articles}
		server := NewServer(&store)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/all")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)

		for _, v := range articles {
			assertContains(t, resp.Body.String(), v.Title)
		}
	})

	t.Run("get pages of articles", func(t *testing.T) {
		progWant, otherWant := MakeSeparatedArticles(20)

		articles := append(progWant, otherWant...)

		store := StubStore{articles: articles}
		server := NewServer(&store)

		cases := []struct {
			path string
			want []Article
		}{
			{"/", progWant[:perPage]},
			{"/page/1", progWant[:perPage]},
			{"/page/2", progWant[len(progWant)-perPage : len(progWant)]},
			{"/page/-5", progWant[:perPage]},
			{"/page/9999", progWant[len(progWant)-perPage : len(progWant)]},
			{"/page/abc", progWant[:perPage]},
			{"/other", otherWant[:perPage]},
			{"/other/page/1", otherWant[:perPage]},
			{"/other/page/2", otherWant[len(otherWant)-perPage : len(otherWant)]},
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
		articles := MakeArticlesOfCategory(10, time.Now(), progCat)

		store := StubStore{articles: articles}
		server := NewServer(&store)

		// Valid article
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/article-1")

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

// Integration test
func TestIntegration(t *testing.T) {
	t.Run("get all", func(t *testing.T) {
		articles := MakeBothTypesOfArticle(20)

		store := InMemStore{articles: articles}
		server := NewServer(&store)

		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/all")

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)

		for _, v := range articles {
			assertContains(t, resp.Body.String(), v.Title)
		}
	})

	t.Run("get pages of articles", func(t *testing.T) {
		progWant, otherWant := MakeSeparatedArticles(20)

		articles := append(progWant, otherWant...)

		store := InMemStore{articles: articles}
		server := NewServer(&store)

		cases := []struct {
			path string
			want []Article
		}{
			{"/", progWant[:perPage]},
			{"/page/1", progWant[:perPage]},
			{"/page/2", progWant[len(progWant)-perPage : len(progWant)]},
			{"/page/-5", progWant[:perPage]},
			{"/page/9999", progWant[len(progWant)-perPage : len(progWant)]},
			{"/page/abc", progWant[:perPage]},
			{"/other", otherWant[:perPage]},
			{"/other/page/1", otherWant[:perPage]},
			{"/other/page/2", otherWant[len(otherWant)-perPage : len(otherWant)]},
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
		articles := MakeArticlesOfCategory(10, time.Now(), progCat)

		store := InMemStore{articles: articles}
		server := NewServer(&store)

		// Valid article
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/article-1")

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
}

func (s *StubStore) getAll() []Article {
	return s.articles
}

func (s *StubStore) getPage(page int, category string) []Article {
	var filtered []Article

	for _, a := range s.getAll() {
		if a.Category == category {
			filtered = append(filtered, a)
		}
	}

	sort.Slice(filtered, func(i int, j int) bool {
		return filtered[i].Published.Before(filtered[j].Published)
	})

	p := page
	if page < 1 {
		p = 1
	}
	maxPage := (len(filtered) / perPage)
	if page > maxPage {
		p = maxPage
	}
	endArticle := (p * perPage) - 1
	if len(filtered) < endArticle {
		endArticle = len(filtered)
	}

	articles := filtered[(p-1)*perPage : endArticle+1]
	return articles
}

func (s *StubStore) getArticle(slug string) Article {
	for _, a := range s.articles {
		if a.Slug == slug {
			return a
		}
	}
	return Article{}
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

func assertArticles(t *testing.T, got, want []Article) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("article slice doesn't match, want %v, got %v", got, want)
	}
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status codes don't match, got %d, want %d", got, want)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s does not contain %s", got, want)
	}
}

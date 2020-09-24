package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestArticle(t *testing.T) {
	t.Run("get all", func(t *testing.T) {
		now := time.Now()

		articles := MakeFakeArticles(2, now, "Programming")

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

		now := time.Now()

		t.Run("with category 'Programming'", func(t *testing.T) {
			articles := MakeFakeArticles(20, now, "Programming")

			store := StubStore{articles: articles}
			server := NewServer(&store)

			cases := []struct {
				path string
				want []Article
			}{
				{"/page/1", articles[:perPage]},
				{"/page/2", articles[len(articles)-perPage : len(articles)]},
				{"/page/-5", articles[:perPage]},
				{"/", articles[:perPage]},
				{"/page/9999", articles[len(articles)-perPage : len(articles)]},
				{"/page/abc", articles[:perPage]},
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

		t.Run("with category 'Other'", func(t *testing.T) {
			articles := MakeFakeArticles(20, now, "Other")

			store := StubStore{articles: articles}
			server := NewServer(&store)

			cases := []struct {
				path string
				want []Article
			}{
				{"/other/page/1", articles[:perPage]},
				{"/other/page/2", articles[len(articles)-perPage : len(articles)]},
				{"/other/page/-5", articles[:perPage]},
				{"/other", articles[:perPage]},
				{"/other/page/9999", articles[len(articles)-perPage : len(articles)]},
				{"/other/page/abc", articles[:perPage]},
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
	})

	t.Run("get view page of single article", func(t *testing.T) {
		now := time.Now()

		articles := MakeFakeArticles(10, now, "Programming")

		store := StubStore{articles: articles}
		server := NewServer(&store)

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

type StubStore struct {
	articles []Article
}

func (s *StubStore) getAll() []Article {
	return s.articles
}

func (s *StubStore) getPage(page int) []Article {
	p := page
	if page < 1 {
		p = 1
	}
	maxPage := (len(s.articles) / perPage)
	if page > maxPage {
		p = maxPage
	}
	endArticle := (p * perPage) - 1
	if len(s.articles) < endArticle {
		endArticle = len(s.articles)
	}

	articles := s.articles[(p-1)*perPage : endArticle+1]
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

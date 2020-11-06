package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var validArticleBase = Article{
	Title:    "I am a valid article!",
	Preview:  "<p>This is a valid preview.</p>",
	Body:     "<p>This is a valid body of an article.</p>",
	Slug:     "this-is_a.v4l1d~slug",
	Category: "Other",
}

func newValidArticleWithTime() Article {
	ret := validArticleBase
	ret.Published = myTimeToString(time.Now().UTC())
	ret.Edited = myTimeToString(time.Now().UTC())
	return ret
}

var editedBase = Article{
	Title:    "Edited article",
	Preview:  "<p>Edited Preview.</p>",
	Body:     "<p>Edited Body.</p>",
	Slug:     "edited-article",
	Category: "Programming",
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

	return s.articles, 0, 0
}

func (s *StubStore) getArticle(slug string) (int, Article) {
	s.calls = append(s.calls, "getArticle")
	for _, a := range s.articles {
		if a.Slug == slug {
			return 1, a
		}
	}
	return 0, Article{}
}

func (s *StubStore) newArticle(a Article) {
	s.calls = append(s.calls, "new")
	s.articles = append(s.articles, a)
}

func (s *StubStore) editArticle(id int, edited Article) {
	s.calls = append(s.calls, "edit")
}

func (s *StubStore) deleteArticle(id int) {
	s.calls = append(s.calls, "delete")
}

func (s *StubStore) doesSlugExist(slug string) bool {
	return false
}

func (s *StubStore) getUser(username, password string) (User, error) {
	return User{}, nil
}

func newGetRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Errorf("Failed to make new GET request, %s", err)
	}
	return req
}

func newPostRequest(t *testing.T, path string, data url.Values) *http.Request {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func newDeleteRequest(t *testing.T, slug string) *http.Request {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "/"+slug+"/delete", nil)
	return req
}

func setDataValues(a Article) url.Values {
	data := url.Values{}
	data.Set("title", a.Title)
	data.Set("preview", a.Preview)
	data.Set("body", a.Body)
	data.Set("slug", a.Slug)
	data.Set("category", a.Category)
	return data
}

func MakeSeparatedArticles(n int) (progWant []Article, otherWant []Article) {
	for i := 1; i <= n; i++ {
		progWant = append(progWant, MakeArticleOfCategory(i, time.Now().UTC(), progCat))
	}
	for i := 1; i <= n; i++ {
		otherWant = append(otherWant, MakeArticleOfCategory(i, time.Now().UTC(), otherCat))
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
	timelessWant := Article{
		Title:    want.Title,
		Preview:  want.Preview,
		Body:     want.Body,
		Slug:     want.Slug,
		Category: want.Category,
	}
	if timelessGot != timelessWant {
		t.Errorf("articles don't match, got %v, want %v", timelessGot, timelessWant)
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
		t.Errorf("got %s, want %s", got, want)
	}
}

func assertNotContain(t *testing.T, got, notWant string) {
	t.Helper()
	if strings.Contains(got, notWant) {
		t.Errorf("got %s, don't want %s", got, notWant)
	}
}

func HashPasswordFast(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 1)
	return string(bytes), err
}

func userData(username, password string) url.Values {
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	return data
}

var pass_hash, _ = HashPasswordFast("password")
var admin = User{
	Username:      "admin",
	Email:         "admin@example.com",
	Password_Hash: pass_hash,
}

func testLogin(t *testing.T, server *Server) {
	data := userData("admin", "password")
	req := newPostRequest(t, "/admin/login", data)
	server.ServeHTTP(httptest.NewRecorder(), req)
}

func testLogout(t *testing.T, server *Server) {
	req := newPostRequest(t, "/admin/logout", nil)
	server.ServeHTTP(httptest.NewRecorder(), req)
}

// Would break if I allowed more than one user, don't copy paste this to other projects.
type StubSessionStore struct {
	// session *sessions.Session
	sesh Sesh
}

func (sss *StubSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	if !sss.sesh.Authenticated {
		return nil, fmt.Errorf("StubSessionStore: auth = false")
	}
	return nil, nil
}

func (sss *StubSessionStore) Set(session *sessions.Session, newSesh Sesh) {
	sss.sesh = newSesh
}

func (sss *StubSessionStore) SaveSession(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return nil
}

func (sss *StubSessionStore) getSesh(s *sessions.Session) Sesh {
	return sss.sesh
}

func (sss *StubSessionStore) SetOption(s *sessions.Session, o string, v interface{}) {

}

func (sss *StubSessionStore) isLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	return sss.sesh.Authenticated
}

func assertLoggedInStatus(t *testing.T, s StubSessionStore, v bool) {
	t.Helper()
	if s.sesh.Authenticated != v {
		t.Error("should be logged in but not")
	}
}

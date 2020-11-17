package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth(t *testing.T) {

	tmpFile, cleanTempFile := makeTempFile()
	defer cleanTempFile()
	store, closeDB := NewFileSystemStore(tmpFile, []Article{}, []User{admin})
	defer closeDB()
	sessStore := StubSessionStore{}
	server := NewServer(store, &sessStore)

	t.Run("200 on GET /admin/login", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/admin/login")
		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 200)
		assertContains(t, resp.Body.String(), "<h1 class=\"title\">Login</h1>")
		assertContains(t, resp.Body.String(), "</form>")
	})

	t.Run("redirect on GET /admin whilst logged out", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := newGetRequest(t, "/admin")
		server.ServeHTTP(resp, req)
		assertStatus(t, resp.Code, http.StatusSeeOther)
	})

	t.Run("successful login attempt", func(t *testing.T) {
		data := userData("admin", "password")

		resp := httptest.NewRecorder()
		req := newPostRequest(t, "/admin/login", data)
		server.ServeHTTP(resp, req)
		assertLoggedInStatus(t, sessStore, true)

		t.Run("redirect to admin panel after login", func(t *testing.T) {
			assertStatus(t, resp.Code, http.StatusSeeOther)
		})

		t.Run("200 on GET /admin", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, "/admin")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 200)
			assertContains(t, resp.Body.String(), "href=\"/admin\">Admin Panel</a>")
			assertContains(t, resp.Body.String(), "<form action=\"/admin/logout\" method=\"post\">")
		})

		t.Run("GET /admin/login redirects to /admin", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newGetRequest(t, "/admin/login")
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, http.StatusSeeOther)
		})

		// logout
		t.Run("POST /logout logs out and redirects to /", func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := newPostRequest(t, "/admin/logout", nil)
			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, http.StatusSeeOther)
			assertLoggedInStatus(t, sessStore, false)

			resp = httptest.NewRecorder()
			req = newGetRequest(t, "/")
			server.ServeHTTP(resp, req)

			assertNotContain(t, resp.Body.String(), "Logged In")
			assertNotContain(t, resp.Body.String(), "Log Out</a>")
		})
	})

	t.Run("401 on POST /admin/login with valid username but invalid password", func(t *testing.T) {
		data := userData("admin", "wrongpassword")

		resp := httptest.NewRecorder()
		req := newPostRequest(t, "/admin/login", data)

		server.ServeHTTP(resp, req)

		assertStatus(t, resp.Code, 401)
		assertContains(t, resp.Body.String(), loginFailed)
	})

	// test each field separately with case table
	t.Run("401 on POST /admin/login with valid credentials but no matching user", func(t *testing.T) {
		cases := []struct {
			username string
			password string
		}{
			{"notadmin", "wrongpassword"},
			{"notadmin", "password"},
		}

		for _, c := range cases {
			data := userData(c.username, c.password)

			resp := httptest.NewRecorder()
			req := newPostRequest(t, "/admin/login", data)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 401)
			assertContains(t, resp.Body.String(), loginFailed)
		}
	})

	t.Run("422 on POST /admin/login with invalid credentials", func(t *testing.T) {
		cases := []struct {
			username string
			password string
			errors   []string
		}{
			{"", "", []string{loginNoUsername, loginNoPassword}},
			{"admin", "", []string{loginNoPassword}},
			{"notadmin", "", []string{loginNoPassword}},
			{"", "password", []string{loginNoUsername}},
			{"", "wrongpassword", []string{loginNoUsername}},
		}

		for _, c := range cases {
			data := userData(c.username, c.password)

			resp := httptest.NewRecorder()
			req := newPostRequest(t, "/admin/login", data)

			server.ServeHTTP(resp, req)

			assertStatus(t, resp.Code, 422)
			for _, err := range c.errors {
				assertContains(t, resp.Body.String(), err)
			}
		}
	})
}

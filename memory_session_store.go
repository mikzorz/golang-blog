package main

import (
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type MemorySessionStore struct {
	cs *sessions.CookieStore
}

func NewMemorySessionStore() *MemorySessionStore {
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	m := new(MemorySessionStore)
	m.cs = m.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	m.cs.Options = &sessions.Options{
		MaxAge:   22800,
		HttpOnly: true,
	}
	return m
}

func (m *MemorySessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return m.cs.Get(r, name)
}

func (m *MemorySessionStore) Set(session *sessions.Session, newSesh Sesh) {
	session.Values["user"] = newSesh
}

func (m *MemorySessionStore) NewCookieStore(keyPairs ...[]byte) *sessions.CookieStore {
	return sessions.NewCookieStore(keyPairs...)
}

func (m *MemorySessionStore) SaveSession(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return s.Save(r, w)
}

func (m *MemorySessionStore) getSesh(s *sessions.Session) Sesh {
	val := s.Values["user"]
	var sesh = Sesh{}
	sesh, ok := val.(Sesh)
	if !ok {
		return Sesh{Authenticated: false}
	}
	return sesh
}

func (m *MemorySessionStore) SetOption(s *sessions.Session, o string, v interface{}) {
	if o == "MaxAge" {
		s.Options.MaxAge = v.(int)
	}
}

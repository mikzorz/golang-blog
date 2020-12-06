package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type Store interface {
	getAll() []Article
	getPage(int, string) ([]Article, int, int)
	getArticle(slug string) (int, Article)
	newArticle(Article)
	editArticle(int, Article)
	deleteArticle(id int)
	doesSlugExist(string) bool
	getUser(username, password string) (User, error)
}

type SessionStore interface {
	Get(r *http.Request, cookie_name string) (*sessions.Session, error)
	Set(*sessions.Session, Sesh)
	SaveSession(*http.Request, http.ResponseWriter, *sessions.Session) error
	getSesh(session *sessions.Session) Sesh
	SetOption(session *sessions.Session, option string, value interface{})
	// isLoggedIn(w http.ResponseWriter, r *http.Request) bool
}

type Server struct {
	store Store
	http.Handler
	sessionStore SessionStore
}

func NewServer(store Store, sessStore SessionStore) *Server {
	s := new(Server)
	s.store = store
	s.sessionStore = sessStore
	gob.Register(Sesh{})

	indexTemplate = setIndexTemplate()
	viewTemplate = setViewTemplate()
	formTemplate = setFormTemplate()
	loginTemplate = setLoginTemplate()
	adminPanelTemplate = setAdminPanelTemplate()

	r := mux.NewRouter()
	r.PathPrefix("/static/css/").Handler(http.StripPrefix("/static/css/", http.FileServer(http.Dir(path.Join(base, "/static/css")))))
	r.PathPrefix("/static/images/").Handler(http.StripPrefix("/static/images/", http.FileServer(http.Dir(path.Join(base, "/static/images")))))

	r.HandleFunc("/", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/new", s.NewArticleForm).Methods("GET")
	r.HandleFunc("/new", s.NewArticle).Methods("POST")
	r.HandleFunc("/page/{page}", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/other", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/other/page/{page}", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/all", s.All).Methods("GET")

	r.HandleFunc("/admin", s.AdminPanel).Methods("GET")
	r.HandleFunc("/admin/login", s.LoginPage).Methods("GET")
	r.HandleFunc("/admin/login", s.AdminLogin).Methods("POST")
	r.HandleFunc("/admin/logout", s.AdminLogout).Methods("POST")

	r.HandleFunc("/{slug}", s.ArticleView).Methods("GET")
	r.HandleFunc("/{slug}/delete", s.DeleteArticle).Methods("GET") // Can't send delete from standard html anchor.
	r.HandleFunc("/{slug}/edit", s.EditArticleForm).Methods("GET")
	r.HandleFunc("/{slug}/edit", s.EditArticle).Methods("POST")

	s.Handler = r

	return s
}

func (s *Server) MainIndexPage(w http.ResponseWriter, r *http.Request) {
	articles, page, maxPage := s.store.getPage(getPageNumber(r), progCat)

	indexPage(w, articles, progCat, page, maxPage, s.isAuth(r))
}

func (s *Server) OtherIndexPage(w http.ResponseWriter, r *http.Request) {
	articles, page, maxPage := s.store.getPage(getPageNumber(r), otherCat)

	indexPage(w, articles, otherCat, page, maxPage, s.isAuth(r))
}

func (s *Server) All(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	// Get articles, then split them into columns.
	articles := articlesWithoutTimes(s.store.getAll())

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	tmpl := indexTemplate
	tmpl.Execute(w, struct {
		Column1     []Article
		Column2     []Article
		Category    string
		LoggedIn    bool
		Dev         bool
		Description string
	}{articles[:len(articles)/2], articles[len(articles)/2:], "", s.isAuth(r), DEV, defaultDescription})
}

func (s *Server) ArticleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	id, article := s.store.getArticle(slug)
	if id > 0 {
		articleView(w, article, s.isAuth(r))
	} else {
		w.WriteHeader(404)
		fmt.Fprint(w, "404 not found")
	}
}

func (s *Server) NewArticleForm(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		executeArticleForm(w, Article{}, template.HTMLAttr(""), "/new", s.isAuth(r))
		return
	} else {
		w.WriteHeader(401)
	}
}

func (s *Server) NewArticle(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		a := getArticleFromForm(r)
		a.Published = myTimeToString(time.Now().UTC())
		a.Edited = a.Published

		errors := s.ValidateArticle(a, true)
		if len(errors) != 0 {
			w.WriteHeader(http.StatusBadRequest)
			executeArticleForm(w, a, template.HTMLAttr("value=\""+a.Slug+"\""), "/new", s.isAuth(r), errors)
			return
		}
		s.store.newArticle(a)
		http.Redirect(w, r, "/all", http.StatusSeeOther)
	} else {
		w.WriteHeader(401)
	}
}

func (s *Server) EditArticleForm(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		vars := mux.Vars(r)
		slug := vars["slug"]
		id, a := s.store.getArticle(slug)
		if id > 0 {
			executeArticleForm(w, a, template.HTMLAttr("value=\""+slug+"\""), "/"+slug+"/edit", s.isAuth(r))
		} else {
			w.WriteHeader(404)
		}
		return
	} else {
		w.WriteHeader(401)
	}
}

func (s *Server) EditArticle(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		vars := mux.Vars(r)
		slug := vars["slug"]
		id, article := s.store.getArticle(slug)
		if id == 0 {
			w.WriteHeader(404)
		} else {
			edit := getArticleFromForm(r)
			edit.Published = article.Published
			edit.Edited = myTimeToString(time.Now().UTC())

			errors := s.ValidateArticle(edit, false)
			if len(errors) != 0 {
				w.WriteHeader(http.StatusBadRequest)
				executeArticleForm(w, edit, template.HTMLAttr("value=\""+edit.Slug+"\""), "/"+article.Slug+"/edit", s.isAuth(r), errors)
				return
			}
			s.store.editArticle(id, edit)
			http.Redirect(w, r, "/"+edit.Slug, http.StatusSeeOther)
		}
	} else {
		w.WriteHeader(401)
	}
}

func (s *Server) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		vars := mux.Vars(r)
		slug := vars["slug"]
		id, _ := s.store.getArticle(slug)
		if id > 0 {
			s.store.deleteArticle(id)
		} else {
			w.WriteHeader(404)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		w.WriteHeader(401)
	}
}

func (s *Server) LoginPage(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
	loginForm(w, nil, s.isAuth(r))
}

func (s *Server) AdminLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := s.sessionStore.Get(r, "user")

	err := r.ParseForm()
	checkErr(err)

	username := r.FormValue("username")
	password := r.FormValue("password")
	if errors := validateUserLogin(username, password); len(errors) != 0 {
		go sendEmailToAdmin(r, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		loginForm(w, errors, s.isAuth(r))
		return
	}
	user, err := s.store.getUser(username, password)
	if err != nil {
		go sendEmailToAdmin(r, false)
		w.WriteHeader(http.StatusUnauthorized)
		loginForm(w, []string{loginFailed}, s.isAuth(r))
		return
	}
	if !user.checkPassword(password) {
		go sendEmailToAdmin(r, false)
		w.WriteHeader(http.StatusUnauthorized)
		loginForm(w, []string{loginFailed}, s.isAuth(r))
		return
	}

	go sendEmailToAdmin(r, true)

	newSesh := Sesh{name: user.Username, Authenticated: true}
	s.sessionStore.Set(session, newSesh)

	err = s.sessionStore.SaveSession(r, w, session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) AdminLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := s.sessionStore.Get(r, "user")

	s.sessionStore.Set(session, Sesh{})
	s.sessionStore.SetOption(session, "MaxAge", -1)

	err := s.sessionStore.SaveSession(r, w, session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) AdminPanel(w http.ResponseWriter, r *http.Request) {
	if s.isAuth(r) {
		adminPanel(w, articlesWithoutTimes(s.store.getAll()), true)
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

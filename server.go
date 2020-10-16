package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/gorilla/mux"
)

type Store interface {
	getAll() []Article
	getPage(int, string) ([]Article, int, int)
	getArticle(string) (int, Article)
	newArticle(Article)
	editArticle(int, Article)
	doesSlugExist(string) bool
}

type Server struct {
	store Store
	http.Handler
}

func NewServer(store Store) *Server {
	s := new(Server)
	s.store = store

	r := mux.NewRouter()
	// r.PathPrefix("/static/css/").Handler(http.StripPrefix("/static/css/", http.FileServer(http.Dir(path.Join(base, "/static/css")))))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(path.Join(base, "/static")))))

	r.HandleFunc("/", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/", s.NewArticle).Methods("POST")
	r.HandleFunc("/new", s.NewArticleForm).Methods("GET")
	r.HandleFunc("/page/{page}", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/other", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/other/page/{page}", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/all", s.All).Methods("GET")
	r.HandleFunc("/{slug}", s.ArticleView).Methods("GET")
	r.HandleFunc("/{slug}", s.EditArticle).Methods("POST") // Cannot send PATCH from html forms
	r.HandleFunc("/{slug}/edit", s.EditArticleForm).Methods("GET")

	s.Handler = r
	return s
}

func (s *Server) MainIndexPage(w http.ResponseWriter, r *http.Request) {
	articles, page, maxPage := s.store.getPage(getPageNumber(r), progCat)

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	indexPage(w, articles, progCat, page, maxPage)
}

func (s *Server) OtherIndexPage(w http.ResponseWriter, r *http.Request) {
	articles, page, maxPage := s.store.getPage(getPageNumber(r), otherCat)

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	indexPage(w, articles, otherCat, page, maxPage)
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
		Column1  []Article
		Column2  []Article
		Category string
	}{articles[:len(articles)/2], articles[len(articles)/2:], ""})
}

func (s *Server) ArticleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	_, article := s.store.getArticle(slug)
	if article != (Article{}) {
		articleView(w, article)
	} else {
		w.WriteHeader(404)
		fmt.Fprint(w, "404 not found")
	}
}

func (s *Server) NewArticleForm(w http.ResponseWriter, r *http.Request) {
	executeArticleForm(w, Article{}, template.HTMLAttr(""), "/")
}

func (s *Server) NewArticle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	checkErr(err)

	a := getArticleFromForm(r)
	a.Published = myTimeToString(time.Now().UTC())
	a.Edited = a.Published

	errors := s.ValidateArticle(a, true)
	if len(errors) != 0 {
		w.WriteHeader(http.StatusBadRequest)
		executeArticleForm(w, a, template.HTMLAttr("value=\""+a.Slug+"\""), "/", errors)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	s.store.newArticle(a)
	s.All(w, r)
}

func (s *Server) EditArticle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	id, article := s.store.getArticle(slug)
	if article == (Article{}) {
		w.WriteHeader(404)
	} else {
		err := r.ParseForm()
		checkErr(err)
		edit := getArticleFromForm(r)
		edit.Published = article.Published
		edit.Edited = myTimeToString(time.Now().UTC())

		errors := s.ValidateArticle(edit, false)
		if len(errors) != 0 {
			w.WriteHeader(http.StatusBadRequest)
			executeArticleForm(w, edit, template.HTMLAttr("value=\""+edit.Slug+"\""), "/"+article.Slug, errors)
			return
		}
		w.WriteHeader(202)
		s.store.editArticle(id, edit)
		articleView(w, edit)
	}
}

func (s *Server) EditArticleForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	_, a := s.store.getArticle(slug)
	if a != (Article{}) {
		executeArticleForm(w, a, template.HTMLAttr("value=\""+slug+"\""), "/"+slug)
	} else {
		w.WriteHeader(404)
	}
}

func indexPage(w http.ResponseWriter, a []Article, cat string, curPage, maxPage int) {
	type ArticleWithIsEdited struct {
		Article
		IsEdited bool
	}

	articlesWithIsEdited := []ArticleWithIsEdited{}
	for _, v := range a {
		isEdited := myStringToTime(v.Published).Before(myStringToTime(v.Edited))
		newA := articleWithoutTime(v)
		articlesWithIsEdited = append(articlesWithIsEdited, ArticleWithIsEdited{newA, isEdited})
	}

	tmpl := indexTemplate
	tmpl.Execute(w, struct {
		Articles []ArticleWithIsEdited
		Category string
		PageInfo PageInfo
	}{articlesWithIsEdited, cat, makePageInfoObject(curPage, maxPage)})
}

func articleView(w http.ResponseWriter, a Article) {
	if DEV {
		viewTemplate = setViewTemplate()
	}
	tmpl := viewTemplate
	tmpl, _ = tmpl.Parse("{{define \"body\"}}" + a.Body + "{{end}}")

	isEdited := myStringToTime(a.Published).Before(myStringToTime(a.Edited))

	tmpl.Execute(w, struct {
		Article  Article
		IsEdited bool
	}{articleWithoutTime(a), isEdited})
}

func executeArticleForm(w http.ResponseWriter, a Article, slugValueAttr template.HTMLAttr, formAction string, errors ...[]string) {
	if DEV {
		formTemplate = setFormTemplate()
	}
	tmpl := formTemplate
	if errors != nil {
		tmpl.Execute(w, struct {
			Article       Article
			SlugValueAttr template.HTMLAttr
			FormAction    string
			Errors        []string
		}{a, slugValueAttr, formAction, errors[0]})
	} else {
		tmpl.Execute(w, struct {
			Article       Article
			SlugValueAttr template.HTMLAttr
			FormAction    string
		}{a, slugValueAttr, formAction})
	}
}

func getArticleFromForm(r *http.Request) Article {
	a := Article{Title: r.FormValue("title")}
	a.Preview = r.FormValue("preview")
	a.Body = r.FormValue("body")
	a.Slug = r.FormValue("slug")
	a.Category = r.FormValue("category")
	return a
}

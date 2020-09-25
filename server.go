package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Article struct {
	Title     string
	Body      string
	Slug      string
	Published time.Time
	Edited    time.Time
	Category  string
}

type Store interface {
	getAll() []Article
	getPage(int, string) []Article
	getArticle(string) Article
}

type Server struct {
	store Store
	http.Handler
}

func NewServer(store Store) *Server {
	s := new(Server)
	s.store = store

	r := mux.NewRouter()

	r.HandleFunc("/", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/page/{page}", s.MainIndexPage).Methods("GET")
	r.HandleFunc("/other", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/other/page/{page}", s.OtherIndexPage).Methods("GET")
	r.HandleFunc("/all", s.All).Methods("GET")
	r.HandleFunc("/{slug}", s.ArticleView).Methods("GET")

	s.Handler = r
	return s
}

func (s *Server) MainIndexPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page, ok := vars["page"]
	if !ok {
		page = "1"
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}
	articles := s.store.getPage(pageInt, progCat)
	fmt.Fprintf(w, "main index, page %s \n", page)
	fmt.Fprint(w, articles)
}

func (s *Server) OtherIndexPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page, ok := vars["page"]
	if !ok {
		page = "1"
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}
	articles := s.store.getPage(pageInt, otherCat)
	fmt.Fprintf(w, "other index, page %s", page)
	fmt.Fprint(w, articles)
}

func (s *Server) All(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprintf(w, "all - %v", s.store.getAll())
}

func (s *Server) ArticleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	article := s.store.getArticle(slug)
	if article != (Article{}) {
		fmt.Fprintf(w, "article view: %s", slug)
		fmt.Fprint(w, article)
	} else {
		w.WriteHeader(404)
		fmt.Fprintf(w, "article view: %s", slug)
	}
}

func MakeBothTypesOfArticle(n int) []Article {
	var articles []Article
	for i := 1; i <= n; i++ {
		articles = append(articles, MakeArticleOfCategory(i, time.Now(), progCat))
		articles = append(articles, MakeArticleOfCategory(i, time.Now(), otherCat))
	}
	return articles
}

func MakeArticlesOfCategory(amount int, now time.Time, category string) []Article {
	ret := []Article{}
	for i := 0; i < amount; i++ {
		art := Article{
			Title:     category + " Article " + strconv.Itoa(i),
			Body:      "Test Article " + strconv.Itoa(i),
			Slug:      "article-" + strconv.Itoa(i),
			Published: now,
			Edited:    now,
			Category:  category,
		}
		ret = append(ret, art)
	}
	return ret
}

func MakeArticleOfCategory(i int, now time.Time, category string) Article {
	return Article{
		Title:     category + " Article " + strconv.Itoa(i),
		Body:      "Test Article " + strconv.Itoa(i),
		Slug:      "article-" + strconv.Itoa(i),
		Published: now,
		Edited:    now,
		Category:  category,
	}
}

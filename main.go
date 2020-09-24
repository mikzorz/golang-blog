package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const port = 3000
const progCat = "Programming"
const otherCat = "Other"

var perPage = 10

func main() {
	now := time.Now()
	fakes := MakeFakeArticles(10, now, progCat)
	fakes = append(fakes, MakeFakeArticles(10, now, otherCat)...)

	store := InMemStore{fakes}
	server := NewServer(&store)

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

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
	getPage(int) []Article
	getArticle(string) Article
}

type InMemStore struct {
	articles []Article
}

func (i *InMemStore) getAll() []Article {
	return []Article{}
}

func (i *InMemStore) getPage(page int) []Article {
	return []Article{}
}

func (i *InMemStore) getArticle(slug string) Article {
	return Article{}
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
	articles := s.store.getPage(pageInt)
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
	articles := s.store.getPage(pageInt)
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

func MakeFakeArticles(amount int, now time.Time, category string) []Article {
	ret := []Article{}
	for i := 0; i < amount; i++ {
		art := Article{
			Title:     category + " Article " + strconv.Itoa(i),
			Body:      "Test Article " + strconv.Itoa(i),
			Slug:      "article-" + strconv.Itoa(i),
			Published: now,
			Edited:    now,
			Category:  category}
		ret = append(ret, art)
	}
	return ret
}

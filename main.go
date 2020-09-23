package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	panic(http.ListenAndServe(":3000", NewServer()))
}

type Server struct {
	http.Handler
}

func NewServer() *Server {
	s := new(Server)
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
	fmt.Fprintf(w, "main index, page %s", page)
}

func (s *Server) OtherIndexPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page, ok := vars["page"]
	if !ok {
		page = "1"
	}
	fmt.Fprintf(w, "other index, page %s", page)
}

func (s *Server) All(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "all")
}

func (s *Server) ArticleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Fprintf(w, "article view: %s", vars["slug"])
}

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Article struct {
	Title     string
	Preview   string
	Body      string
	Slug      string
	Published string
	Edited    string
	Category  string
}

type PageInfo struct {
	CurrentPage int
	MaxPage     int
	Next        int
	Prev        int
}

type Store interface {
	getAll() []Article
	getPage(int, string) ([]Article, int, int)
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
	// r.PathPrefix("/static/css/").Handler(http.StripPrefix("/static/css/", http.FileServer(http.Dir(path.Join(base, "/static/css")))))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(path.Join(base, "/static")))))

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
	articles, page, maxPage := s.store.getPage(getPageNumber(r), progCat)

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	tmpl := indexTemplate
	tmpl.Execute(w, struct {
		Articles []Article
		Category string
		PageInfo PageInfo
	}{articles, progCat, makePageInfoObject(page, maxPage)})
}

func (s *Server) OtherIndexPage(w http.ResponseWriter, r *http.Request) {
	articles, page, maxPage := s.store.getPage(getPageNumber(r), otherCat)

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	tmpl := indexTemplate
	tmpl.Execute(w, struct {
		Articles []Article
		Category string
		PageInfo PageInfo
	}{articles, otherCat, makePageInfoObject(page, maxPage)})
}

func (s *Server) All(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	// Get articles, then split them into columns.
	articles := s.store.getAll()

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}
	tmpl := indexTemplate
	err := tmpl.Execute(w, struct {
		Column1  []Article
		Column2  []Article
		Category string
	}{articles[:len(articles)/2], articles[len(articles)/2:], ""})
	if err != nil {
		log.Printf("%s", err)
	}
}

func (s *Server) ArticleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	article := s.store.getArticle(slug)
	if article != (Article{}) {
		tmpl := template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/article.html"))
		tmpl, _ = tmpl.Parse("{{define \"body\"}}" + article.Body + "{{end}}")

		tmpl.Execute(w, struct{ Article Article }{article})
	} else {
		w.WriteHeader(404)
		fmt.Fprint(w, "404 not found")
	}
}

func getPageNumber(r *http.Request) int {
	vars := mux.Vars(r)
	page, ok := vars["page"]
	if !ok {
		page = "1"
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}
	return pageInt
}

func MakeBothTypesOfArticle(n int) []Article {
	var articles []Article
	for i := 1; i <= n; i++ {
		articles = append(articles, MakeArticleOfCategory(i, time.Now(), progCat))
		articles = append(articles, MakeArticleOfCategory(i, time.Now(), otherCat))
	}
	return articles
}

// Only used in tests. Should probably move to test file.
func MakeArticlesOfCategory(amount int, now time.Time, category string) []Article {
	ret := []Article{}
	for i := 0; i < amount; i++ {
		art := Article{
			Title:     category + " Article " + strconv.Itoa(i),
			Body:      "Test Article " + strconv.Itoa(i),
			Slug:      strings.ToLower(category) + "-article-" + strconv.Itoa(i),
			Published: myTimeToString(now.UTC().Add(time.Duration(i))),
			Edited:    myTimeToString(now.UTC().Add(time.Duration(i))),
			Category:  category,
		}
		ret = append(ret, art)
	}
	return ret
}

func MakeArticleOfCategory(i int, now time.Time, category string) Article {
	ret := Article{
		Title:   category + " Article " + strconv.Itoa(i),
		Preview: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.",
		Body: `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>`,
		Slug:      strings.ToLower(category) + "-article-" + strconv.Itoa(i),
		Published: myTimeToString(now.UTC().Add(time.Duration(i))),
		Edited:    myTimeToString(now.UTC().Add(time.Duration(i))),
		Category:  category,
	}
	return ret
}

func makePageInfoObject(page, maxPage int) PageInfo {
	return PageInfo{page, maxPage, page + 1, page - 1}
}

func setIndexTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/index.html"))
}

func myTimeToString(t time.Time) string {
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	dateString := strconv.Itoa(year) + "-" +
		fmt.Sprintf("%02d", int(month)) + "-" +
		fmt.Sprintf("%02d", day) + " " +
		fmt.Sprintf("%02d", hour) + ":" +
		fmt.Sprintf("%02d", minute) + ":" +
		fmt.Sprintf("%02d", second)
	return dateString
}

func myStringToTime(s string) time.Time {
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, s)
	checkErr(err)
	return t
}

// I made these, thinking that they would reduce the amount of data sent to the user.
// Then I realized that everything is rendered server-side.
// If it's not rendered in the template, it isn't being sent...
func articlesWithoutBodies(articles []Article) []Article {
	ret := make([]Article, len(articles))
	for i, a := range articles {
		ret[i] = articleWithoutBody(a)
	}
	return ret
}

func articleWithoutBody(a Article) Article {
	ret := a
	ret.Body = ""
	return ret
}

func checkErr(err error) {
	if err != nil {
		// log.Fatal(err)
		panic(err)
	}
}

func makeTempFile() (*os.File, func()) {
	tmpfile, err := ioutil.TempFile("", "db")

	if err != nil {
		log.Fatalf("could not create temp file %v", err)
	}

	tmpfile.Write([]byte(""))

	removeFile := func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}

	return tmpfile, removeFile
}

// Given a slice of articles and a page number, will return that page's articles, the actual current page and the highest page number.
func paginate(a []Article, page int) ([]Article, int, int) {
	p := page
	if page < 1 {
		p = 1
	}
	maxPage := (len(a) / perPage)
	if page > maxPage {
		p = maxPage
	}
	endArticle := (p * perPage) - 1
	if len(a) < endArticle {
		endArticle = len(a)
	}

	return a[(p-1)*perPage : endArticle+1], p, maxPage
}

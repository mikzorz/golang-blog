package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var indexTemplate *template.Template
var viewTemplate *template.Template
var formTemplate *template.Template
var loginTemplate *template.Template
var adminPanelTemplate *template.Template

const perPage = 10

const maxTitleLength = 50
const progCat = "Programming"
const otherCat = "Other"

const (
	errTitleLong         = "Title is too long"
	errTitleEmpty        = "Title cannot be empty"
	errPreviewEmpty      = "Preview cannot be empty"
	errBodyEmpty         = "Body cannot be empty"
	errSlugEmpty         = "Slug cannot be empty"
	errSlugAlreadyExists = "Slug is already being used by another article"
	errSlugBad           = "Slug contains illegal characters"
	errCatInvalid        = "Category is invalid"
)

const (
	loginNoUsername = "Please enter a username."
	loginNoPassword = "Please enter a password."
	loginFailed     = "Incorrect username and/or password. Try again."
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

type User struct {
	Id            int
	Username      string
	Email         string
	Password_Hash string
}

type Sesh struct {
	name          string
	Authenticated bool
}

func (u *User) checkPassword(password string) bool {
	// hash password then check if == to Password_Hash
	err := bcrypt.CompareHashAndPassword([]byte(u.Password_Hash), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func isEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	if !emailRegex.MatchString(e) {
		return false
	}
	parts := strings.Split(e, "@")
	mx, err := net.LookupMX(parts[1])
	if err != nil || len(mx) == 0 {
		return false
	}
	return true
}

func (s *Server) isAuth(r *http.Request) bool {
	session, err := s.sessionStore.Get(r, "user")
	if err != nil {
		return false
	}
	sesh := s.sessionStore.getSesh(session)
	return sesh.Authenticated
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func MakeBothTypesOfArticle(n int) []Article {
	var articles []Article
	for i := 1; i <= n; i++ {
		articles = append(articles, MakeArticleOfCategory(i, time.Now().UTC(), progCat))
		articles = append(articles, MakeArticleOfCategory(i, time.Now().UTC(), otherCat))
	}
	return articles
}

// Only used in tests. Could be moved to test file.
func MakeArticlesOfCategory(amount int, now time.Time, category string) []Article {
	ret := []Article{}
	for i := 0; i < amount; i++ {
		nowOffset := myTimeToString(now.Add(time.Hour * -1).Add(time.Second * time.Duration(i)))
		art := Article{
			Title:     category + " Article " + strconv.Itoa(i),
			Body:      "Test Article " + strconv.Itoa(i),
			Slug:      strings.ToLower(category) + "-article-" + strconv.Itoa(i),
			Published: nowOffset,
			Edited:    nowOffset,
			Category:  category,
		}
		ret = append(ret, art)
	}
	return ret
}

func MakeArticleOfCategory(i int, now time.Time, category string) Article {
	nowOffset := myTimeToString(now.Add(time.Hour * -1).Add(time.Second * time.Duration(i)))
	ret := Article{
		Title:   category + " Article " + strconv.Itoa(i),
		Preview: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.",
		Body: `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec in tincidunt magna. Maecenas venenatis dictum porttitor. Nulla condimentum est odio, ac blandit lorem posuere quis. Donec bibendum lectus nec ligula laoreet, a varius mi blandit. Fusce vel consequat odio. Praesent porttitor odio vel tincidunt sodales.</p>`,
		Slug:      strings.ToLower(category) + "-article-" + strconv.Itoa(i),
		Published: nowOffset,
		Edited:    nowOffset,
		Category:  category,
	}
	return ret
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

func articlesWithoutTimes(articles []Article) []Article {
	ret := make([]Article, len(articles))
	for i, a := range articles {
		ret[i] = articleWithoutTime(a)
	}
	return ret
}

func articleWithoutTime(a Article) Article {
	ret := a
	ret.Published = ret.Published[:10]
	ret.Edited = ret.Edited[:10]
	return ret
}

func checkErr(err error) {
	if err != nil {
		log.Print(err.Error())
		// panic(err)
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

func reverseArticles(in []Article) []Article {
	ret := make([]Article, len(in))

	for i, a := range in {
		ret[len(ret)-i-1] = a
	}

	return ret
}

func (s *Server) ValidateArticle(a Article, checkSlugExists bool) (errors []string) {
	// Check each field.
	// If Title is too long or doesn't exist.
	if len([]rune(a.Title)) > maxTitleLength {
		errors = append(errors, errTitleLong)
	}
	if len(a.Title) == 0 {
		errors = append(errors, errTitleEmpty)
	}
	// If Preview is empty.
	if len(a.Preview) == 0 {
		errors = append(errors, errPreviewEmpty)
	}
	// If Body is empty.
	if len(a.Body) == 0 {
		errors = append(errors, errBodyEmpty)
	}
	// If Slug contains non-valid characters.
	if len(a.Slug) == 0 {
		errors = append(errors, errSlugEmpty)
	}
	// If Slug is already in use.
	if checkSlugExists && s.store.doesSlugExist(a.Slug) {
		errors = append(errors, errSlugAlreadyExists)
	}
	illegalChars := "&$+,/:;=?@# <>[]{}|\\^%"
slugCheck:
	for i := 0; i < len(a.Slug); i++ {
		for j := 0; j < len(illegalChars); j++ {
			if a.Slug[i] == illegalChars[j] {
				errors = append(errors, errSlugBad)
				break slugCheck
			}
		}
	}
	// If Category is not one of the valid categories.
	if a.Category != progCat && a.Category != otherCat {
		errors = append(errors, errCatInvalid)
	}
	return
}

func validateUserLogin(username, password string) []string {
	ret := []string{}
	if username == "" {
		ret = append(ret, loginNoUsername)
	}
	if password == "" {
		ret = append(ret, loginNoPassword)
	}
	return ret
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

func makePageInfoObject(page, maxPage int) PageInfo {
	return PageInfo{page, maxPage, page + 1, page - 1}
}

func setIndexTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/index.html"))
}

func setViewTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/article.html"))
}

func setFormTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/articleForm.html"))
}

func setLoginTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/login.html"))
}

func setAdminPanelTemplate() *template.Template {
	return template.Must(template.ParseFiles("static/templates/base.html", "static/templates/nav.html", "static/templates/adminPanel.html"))
}

func getArticleFromForm(r *http.Request) Article {
	err := r.ParseForm()
	checkErr(err)

	a := Article{Title: r.FormValue("title")}
	a.Preview = r.FormValue("preview")
	a.Body = r.FormValue("body")
	a.Slug = r.FormValue("slug")
	a.Category = r.Form["category"][0]
	return a
}

func indexPage(w http.ResponseWriter, a []Article, cat string, curPage, maxPage int, loggedIn bool) {
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

	// Reload HTML without rebuilding project.
	if DEV {
		indexTemplate = setIndexTemplate()
	}

	tmpl := indexTemplate
	tmpl.Execute(w, struct {
		Articles []ArticleWithIsEdited
		Category string
		PageInfo PageInfo
		LoggedIn bool
		Dev      bool
	}{articlesWithIsEdited, cat, makePageInfoObject(curPage, maxPage), loggedIn, DEV})
}

func articleView(w http.ResponseWriter, a Article, loggedIn bool) {
	viewTemplate = setViewTemplate()

	tmpl := viewTemplate
	// This could be done differently. This may also be what was breaking the 'if DEV' statement.
	tmpl = template.Must(tmpl.Parse("{{define \"body\"}}" + a.Body + "{{end}}"))

	isEdited := myStringToTime(a.Published).Before(myStringToTime(a.Edited))

	tmpl.Execute(w, struct {
		Article  Article
		IsEdited bool
		LoggedIn bool
		Dev      bool
	}{articleWithoutTime(a), isEdited, loggedIn, DEV})
}

func executeArticleForm(w http.ResponseWriter, a Article, slugValueAttr template.HTMLAttr, formAction string, loggedIn bool, errors ...[]string) {
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
			LoggedIn      bool
			Dev           bool
		}{a, slugValueAttr, formAction, errors[0], loggedIn, DEV})
	} else {
		tmpl.Execute(w, struct {
			Article       Article
			SlugValueAttr template.HTMLAttr
			FormAction    string
			Errors        []string
			Dev           bool
			LoggedIn      bool
		}{a, slugValueAttr, formAction, []string{}, loggedIn, DEV})
	}
}

func loginForm(w http.ResponseWriter, errors []string, loggedIn bool) {
	if DEV {
		loginTemplate = setLoginTemplate()
	}
	tmpl := loginTemplate
	tmpl.Execute(w, struct {
		Errors   []string
		LoggedIn bool
		Dev      bool
	}{errors, loggedIn, DEV})
}

func adminPanel(w http.ResponseWriter, articles []Article, loggedIn bool) {
	if DEV {
		adminPanelTemplate = setAdminPanelTemplate()
	}
	tmpl := adminPanelTemplate
	tmpl.Execute(w, struct {
		Articles []Article
		LoggedIn bool
		Dev      bool
	}{articles, loggedIn, DEV})
}

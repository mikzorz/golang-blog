package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

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

func MakeBothTypesOfArticle(n int) []Article {
	var articles []Article
	for i := 1; i <= n; i++ {
		articles = append(articles, MakeArticleOfCategory(i, time.Now().UTC(), progCat))
		articles = append(articles, MakeArticleOfCategory(i, time.Now().UTC(), otherCat))
	}
	return articles
}

// Only used in tests. Should probably move to test file.
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
	maxTitleLength := 50 // Not finalized, test in browser to see when titles look bad.
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
	return template.Must(template.ParseFiles("static/templates/articleForm.html"))
}

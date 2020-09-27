package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
)

const DEV = true

var base = os.Getenv("GOPATH") + "/src/github.com/mikzorz/blog"
var indexTemplate = setIndexTemplate()

const port = 3000
const progCat = "Programming"
const otherCat = "Other"

var perPage = 10

func main() {
	fakes := MakeBothTypesOfArticle(100)

	store := InMemStore{fakes}
	server := NewServer(&store)

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

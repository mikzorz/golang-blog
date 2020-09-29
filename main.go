package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
)

const DEV = true

var base = os.Getenv("GOPATH") + "/src/github.com/mikzorz/blog"
var dbFileName = base + "blog.db"
var indexTemplate = setIndexTemplate()

const port = 3000
const progCat = "Programming"
const otherCat = "Other"

var perPage = 10

func main() {
	var dbFile *os.File
	var err error

	if DEV {
		devDB, cleanDev := makeTempFile()
		defer cleanDev()
		dbFile = devDB
	} else {
		dbFile, err = os.OpenFile(dbFileName, os.O_RDWR|os.O_CREATE, 0600)

		if err != nil {
			log.Fatalf("problem opening %s %v", dbFileName, err)
		}
	}

	fakes := MakeBothTypesOfArticle(100)

	store, closeDB := NewFileSystemStore(dbFile, fakes)
	defer closeDB()
	server := NewServer(store)

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

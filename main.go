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
var viewTemplate = setViewTemplate()
var formTemplate = setFormTemplate()
var loginTemplate = setLoginTemplate()
var adminPanelTemplate = setAdminPanelTemplate()

const port = 3000
const progCat = "Programming"
const otherCat = "Other"

var perPage = 10

func main() {
	var dbFile *os.File
	var err error
	var server *Server

	if DEV {
		devDB, cleanDev := makeTempFile()
		defer cleanDev()
		dbFile = devDB

		fakes := MakeBothTypesOfArticle(100)

		pass_hash, _ := HashPasswordFast("password")
		admin := User{
			Username:      "admin",
			Email:         "admin@example.com",
			Password_Hash: pass_hash,
		}

		store, closeDB := NewFileSystemStore(dbFile, fakes, []User{admin})
		defer closeDB()
		sessStore := NewMemorySessionStore()
		server = NewServer(store, sessStore)
	} else {
		dbFile, err = os.OpenFile(dbFileName, os.O_RDWR|os.O_CREATE, 0600)

		if err != nil {
			log.Fatalf("problem opening %s %v", dbFileName, err)
		}
	}

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

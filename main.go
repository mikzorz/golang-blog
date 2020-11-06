package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
)

const DEV = false
const port = 3000

var base = os.Getenv("GOPATH") + "/src/github.com/mikzorz/blog"
var dbFileName = base + "blog.db"

func main() {
	var dbFile *os.File
	var err error
	var server *Server

	if DEV {
		log.Print("Running in DEVELOPMENT mode.")

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
		log.Print("Running in PRODUCTION mode.")

		dbFile, err = os.OpenFile(dbFileName, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Fatalf("problem opening %s %v", dbFileName, err)
		}
		defer dbFile.Close()

		username := os.Getenv("blog_username")
		if username == "" {
			log.Fatal("Environment variable not set: blog_username")
		}
		email := os.Getenv("blog_email")
		if email == "" {
			log.Fatal("Environment variable not set: blog_email")
		}
		password := os.Getenv("blog_password")
		if password == "" {
			log.Fatal("Environment variable not set: blog_password")
		}
		pass_hash, err := HashPassword(password)
		if err != nil {
			log.Fatal("Couldn't hash password")
		}

		admin := User{
			Username:      username,
			Email:         email,
			Password_Hash: pass_hash,
		}

		store, closeDB := NewFileSystemStore(dbFile, []Article{}, []User{admin})
		defer closeDB()
		sessStore := NewMemorySessionStore()
		server = NewServer(store, sessStore)
	}

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

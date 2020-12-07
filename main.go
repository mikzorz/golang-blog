package main

import (
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"

	"github.com/jordan-wright/email"
)

var DEV bool

const port = 3000

var base = os.Getenv("blog_dir")

// Default
var dbName = "blog.db"
var dbPath = base + "/" + dbName

var sendEmailToAdmin func(*http.Request, bool) = func(*http.Request, bool) {}

// Defaults for testing. Uses env vars for production.
var admin_username = "admin"
var admin_pass = "password"

func main() {
	if base == "" {
		log.Fatal("Environment variable not set: blog_dir")
	}

	var err error
	var dbFile *os.File
	var server *Server

	DEV, err = strconv.ParseBool(os.Getenv("blog_dev"))
	if err != nil {
		log.Print("Environment variable not set: blog_dev. Defaulting to FALSE")
		DEV = false
	}

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

		customDbPath := os.Getenv("blog_db")
		if customDbPath == "" {
			log.Print("Environment variable not set: blog_db, will save db as $PWD/blog.db")
		} else {
			dbPath = customDbPath
		}

		dbFile, err = os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Fatalf("problem opening %s %v", dbPath, err)
		}
		defer dbFile.Close()

		admin_username = os.Getenv("blog_username")
		if admin_username == "" {
			log.Fatal("Environment variable not set: blog_username")
		}

		admin_pass = os.Getenv("blog_password")
		if admin_pass == "" {
			log.Fatal("Environment variable not set: blog_password")
		}

		// Not used at the moment.
		// pass_hash, err := HashPassword(admin_pass)
		// if err != nil {
		// 	log.Fatal("Couldn't hash password")
		// }

		admin_email := os.Getenv("blog_email")
		if admin_email == "" {
			log.Fatal("Environment variable not set: blog_email")
		}
		if !isEmailValid(admin_email) {
			log.Fatal("Environment variable invalid: blog_email")
		}

		bridge_pass := os.Getenv("blog_bridgepass")
		if bridge_pass == "" {
			log.Fatal("Environment variable not set: blog_bridgepass")
		}

		// admin := User{
		// 	Username:      admin_username,
		// 	Email:         admin_email,
		// 	Password_Hash: pass_hash,
		// }

		// Only used in production mode. Not in tests nor development mode.
		sendEmailToAdmin = func(r *http.Request, successfulLogin bool) {
			e := email.NewEmail()
			e.From = "Blog Server <" + admin_email + ">"
			e.To = []string{admin_email}

			// if backup {
			// TRYING TO ATTACH A FILE CAUSES A 'ContentID is not valid' error.
			// e.Subject = "Blog Backup"
			// e.AttachFile(dbPath)

			// Send details about the login attempt. Location, username.
			ip := getIP(r)
			if successfulLogin {
				e.Subject = "Blog Successful Login Attempt Notification"
				e.Text = []byte("SUCCESSFUL LOGIN ATTEMPT. IP: " + ip)
			} else {
				e.Subject = "Blog Failed Login Attempt Notification"
				e.Text = []byte("FAILED LOGIN ATTEMPT. IP: " + ip + ", attempted username: " + r.FormValue("username") + ", attempted password: " + r.FormValue("password"))
			}

			err := e.Send("127.0.0.1:1025", smtp.PlainAuth("", admin_email, bridge_pass, "127.0.0.1"))
			if err != nil {
				log.Print(err)
			}
		}

		// store, closeDB := NewFileSystemStore(dbFile, []Article{}, []User{admin})
		store, closeDB := NewFileSystemStore(dbFile, []Article{}, []User{User{}})
		defer closeDB()
		sessStore := NewMemorySessionStore()
		server = NewServer(store, sessStore)
	}

	log.Printf("Running server on port %d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), server); err != nil {
		log.Fatalf("could not listen on port %d %v", port, err)
	}
}

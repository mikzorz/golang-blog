package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type FileSystemStore struct {
	db *sql.DB
}

func NewFileSystemStore(dbFile *os.File, articles []Article, users []User) (*FileSystemStore, func()) {
	f := new(FileSystemStore)

	if dbFile == nil {
		log.Print("nil file given to NewFileSystemStore")
	} else {
		db, err := sql.Open("sqlite3", dbFile.Name())
		checkErr(err)
		f.db = db

		// If dbFile is empty, setup db.
		f.setupDB(dbFile)

		if users != nil {
			f.saveUsers(users)
		}

		if articles != nil {
			if len(f.getAll()) == 0 {
				f.saveArticles(articles)
			}
		} else {
			// log.Print("no articles given to NewFileSystemStore")
		}
	}

	cleanUp := func() {
		f.db.Close()
	}
	return f, cleanUp
}

func (f *FileSystemStore) getAll() []Article {
	var ret []Article
	rows, err := f.db.Query("SELECT * FROM Articles")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var a Article
		var id int
		err = rows.Scan(&id, &a.Title, &a.Preview, &a.Body, &a.Slug, &a.Published, &a.Edited, &a.Category)
		checkErr(err)
		ret = append(ret, a)
	}

	return reverseArticles(ret)
}

func (f *FileSystemStore) getPage(page int, category string) (articles []Article, p int, maxPage int) {
	var ret []Article
	rows, err := f.db.Query("SELECT * FROM Articles WHERE Category = ?", category)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var a Article
		var id int
		err = rows.Scan(&id, &a.Title, &a.Preview, &a.Body, &a.Slug, &a.Published, &a.Edited, &a.Category)
		checkErr(err)
		ret = append(ret, a)
	}

	return f.paginate(reverseArticles(ret), page)
}

func (f *FileSystemStore) getArticle(slug string) (int, Article) {
	rows, err := f.db.Query("SELECT * FROM Articles WHERE Slug = ? Limit 1", strings.ToLower(slug))
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var a Article
		var id int
		err = rows.Scan(&id, &a.Title, &a.Preview, &a.Body, &a.Slug, &a.Published, &a.Edited, &a.Category)
		checkErr(err)
		return id, a
	}
	return 0, Article{}
}

func (f *FileSystemStore) newArticle(a Article) {
	stmt, err := f.db.Prepare("INSERT INTO Articles(Title, Preview, Body, Slug, Published, Edited, Category) values(?, ?, ?, ?, DATETIME(?), ?, ?)")
	checkErr(err)
	_, err = stmt.Exec(a.Title, a.Preview, a.Body, strings.ToLower(a.Slug), a.Published, a.Edited, a.Category)
	checkErr(err)
}

func (f *FileSystemStore) editArticle(id int, edited Article) {
	stmt, err := f.db.Prepare("UPDATE Articles SET Title = ?, Preview = ?, Body = ?, Slug = ?, Edited = ?, Category = ? WHERE uid = ?")
	checkErr(err)
	_, err = stmt.Exec(edited.Title, edited.Preview, edited.Body, edited.Slug, edited.Edited, edited.Category, id)
	checkErr(err)
}

func (f *FileSystemStore) deleteArticle(id int) {
	stmt, err := f.db.Prepare("DELETE FROM Articles WHERE uid = ?")
	checkErr(err)
	_, err = stmt.Exec(id)
	checkErr(err)
}

func (f *FileSystemStore) saveArticles(articles []Article) {
	for _, a := range articles {
		f.newArticle(a)
	}
}

func (f *FileSystemStore) doesSlugExist(slug string) bool {
	_, a := f.getArticle(strings.ToLower(slug))
	if a == (Article{}) {
		return false
	}
	return true
}

// func (f *FileSystemStore) getIdFromSlug(slug string) int {
// 	return 0
// }

// User

func (f *FileSystemStore) newUser(u User) {
	stmt, err := f.db.Prepare("INSERT INTO Users(Username, Email, Password_Hash) values(?, ?, ?)")
	checkErr(err)
	_, err = stmt.Exec(u.Username, u.Email, u.Password_Hash)
	checkErr(err)
}

func (f *FileSystemStore) saveUsers(users []User) {
	for _, u := range users {
		f.newUser(u)
	}
}

func (f *FileSystemStore) getUser(username string) (User, error) {
	rows, err := f.db.Query("SELECT * FROM Users WHERE Username = ? Limit 1", username)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var u User
		err = rows.Scan(&u.Id, &u.Username, &u.Email, &u.Password_Hash)
		checkErr(err)
		return u, nil
	}
	return User{}, fmt.Errorf("user does not exist")
}

func (f *FileSystemStore) setupDB(dbFile *os.File) {
	fileInfo, err := dbFile.Stat()
	checkErr(err)
	if fileInfo.Size() == 0 {
		f.createTables(f.db)
	}
}

func (f *FileSystemStore) createTables(db *sql.DB) {
	createArticleTableSQL := `CREATE TABLE Articles (
    "uid" INTEGER PRIMARY KEY AUTOINCREMENT,
    "Title" VARCHAR(64) NULL,
    "Preview" TEXT NULL,
    "Body" TEXT NULL,
    "Slug" VARCHAR(64) NULL,
    "Published" VARCHAR(64) NULL,
    "Edited" VARCHAR(64) NULL,
    "Category" VARCHAR(64) NULL
  );`
	stmt, err := db.Prepare(createArticleTableSQL)
	checkErr(err)
	stmt.Exec()

	createUserTableSQL := `CREATE TABLE Users (
		"uid" INTEGER PRIMARY KEY AUTOINCREMENT,
		"Username" VARCHAR(64) NULL,
		"Email" VARCHAR(64) NULL,
		"Password_Hash" VARCHAR(255) NULL
	);`
	stmt, err = db.Prepare(createUserTableSQL)
	checkErr(err)
	stmt.Exec()
}

// Given a slice of articles and a page number, will return that page's articles, the actual current page and the highest page number.
func (f *FileSystemStore) paginate(a []Article, page int) ([]Article, int, int) {
	if len(a) <= perPage {
		return a, 1, 1
	}

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

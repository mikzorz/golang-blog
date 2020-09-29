package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type FileSystemStore struct {
	db *sql.DB
}

func NewFileSystemStore(dbFile *os.File, articles ...[]Article) (*FileSystemStore, func()) {
	f := new(FileSystemStore)

	if dbFile == nil {
		log.Print("nil file given to NewFileSystemStore")
	} else {
		db, err := sql.Open("sqlite3", dbFile.Name())
		checkErr(err)
		f.db = db

		// If dbFile is empty, setup db.
		f.setupDB(dbFile)

		if articles != nil {
			log.Print("articles given to NewFileSystemStore")
			f.saveArticles(articles[0])
		} else {
			log.Print("no articles given to NewFileSystemStore")
		}
	}

	cleanUp := func() {
		f.db.Close()
	}
	return f, cleanUp
}

func (f *FileSystemStore) getAll() []Article {
	var ret []Article
	rows, err := f.db.Query("SELECT * FROM Article")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var a Article
		var id int
		err = rows.Scan(&id, &a.Title, &a.Preview, &a.Body, &a.Slug, &a.Published, &a.Edited, &a.Category)
		checkErr(err)
		ret = append(ret, a)
	}

	return ret
}

func (f *FileSystemStore) getPage(page int, category string) (articles []Article, p int, maxPage int) {
	var ret []Article
	rows, err := f.db.Query("SELECT * FROM Article WHERE Category = ?", category)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var a Article
		var id int
		err = rows.Scan(&id, &a.Title, &a.Preview, &a.Body, &a.Slug, &a.Published, &a.Edited, &a.Category)
		checkErr(err)
		ret = append(ret, a)
	}

	return paginate(ret, page)
}

func (f *FileSystemStore) getArticle(slug string) Article {
	return Article{}
}

func (f *FileSystemStore) saveArticles(articles []Article) {
	stmt, err := f.db.Prepare("INSERT INTO Article(Title, Preview, Body, Slug, Published, Edited, Category) values(?, ?, ?, ?, ?, ?, ?)")
	checkErr(err)

	for _, a := range articles {
		_, err = stmt.Exec(a.Title, a.Preview, a.Body, a.Slug, a.Published, a.Edited, a.Category)
	}
	checkErr(err)
}

func (f *FileSystemStore) setupDB(dbFile *os.File) {
	fileInfo, err := dbFile.Stat()
	checkErr(err)
	if fileInfo.Size() == 0 {
		f.createTable(f.db)
	}
}

func (f *FileSystemStore) createTable(db *sql.DB) {
	createArticleTableSQL := `CREATE TABLE Article (
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
}

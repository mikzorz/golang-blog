package main

import (
	"testing"
	"time"
)

// Store test
func TestFileSystemStore(t *testing.T) {
	t.Run("load store", func(t *testing.T) {
		t.Run("saves and doesn't overwrite when calling NewFileSystemStore", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			articles := MakeBothTypesOfArticle(25)
			_, closeDB := NewFileSystemStore(tmpFile, articles)
			closeDB()

			// Normal load, check if 50 articles exist
			store, closeDB := NewFileSystemStore(tmpFile)

			got := store.getAll()

			if len(got) != 50 {
				t.Error("Articles not saved between DB reloads.")
			}

			closeDB()

			// Because tmpFile is not empty, ignore articles, check if 50 articles exist
			articles = MakeBothTypesOfArticle(25)
			store, closeDB = NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got = store.getAll()

			if len(got) != 50 {
				t.Error("Articles were added to a non-empty db file during db reload")
			}
		})
		t.Run("can load dbfile with tables but no articles", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			_, closeDB := NewFileSystemStore(tmpFile)
			closeDB()
			// Blank db setup finished.

			articles := MakeBothTypesOfArticle(25)
			store, closeDB := NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got := store.getAll()

			if len(got) != 50 {
				t.Error("failed to write to an already setup empty db.")
			}
		})
	})

	t.Run("new store", func(t *testing.T) {
		t.Run("get all", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			articles := MakeBothTypesOfArticle(50)
			store, closeDB := NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			got := store.getAll()
			want := reverseArticles(articles)
			assertArticles(t, got, want)
		})

		t.Run("get page", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			progWant, otherWant := MakeSeparatedArticles(50)
			store, closeDB := NewFileSystemStore(tmpFile, append(progWant, otherWant...))
			defer closeDB()

			progWant = reverseArticles(progWant)
			otherWant = reverseArticles(otherWant)

			got, p, mxP := store.getPage(1, progCat)

			assertInt(t, p, 1)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, progWant[0:perPage])

			got, p, mxP = store.getPage(-1, progCat)

			assertInt(t, p, 1)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, progWant[0:perPage])

			got, p, mxP = store.getPage(3, otherCat)

			assertInt(t, p, 3)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, otherWant[2*perPage:3*perPage])

			got, p, mxP = store.getPage(6, otherCat)

			assertInt(t, p, 5)
			assertInt(t, mxP, 50/perPage)
			assertArticles(t, got, otherWant[4*perPage:5*perPage])
		})

		t.Run("get single article", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			prog, other := MakeSeparatedArticles(20)
			store, closeDB := NewFileSystemStore(tmpFile, append(prog, other...))
			defer closeDB()

			_, got := store.getArticle(prog[0].Slug)
			assertArticle(t, got, prog[0])

			_, got = store.getArticle("does-not-exist")
			assertArticle(t, got, Article{})
		})

		t.Run("save new article", func(t *testing.T) {
			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			store, closeDB := NewFileSystemStore(tmpFile)
			defer closeDB()

			validArticle := validArticleBase
			validArticle.Published = myTimeToString(time.Now().UTC())
			validArticle.Edited = myTimeToString(time.Now().UTC())

			store.newArticle(validArticle)

			_, got := store.getArticle(validArticle.Slug)

			assertArticle(t, got, validArticle)
		})

		t.Run("edit article", func(t *testing.T) {
			old := validArticleBase
			old.Published = myTimeToString(time.Now().UTC())
			old.Edited = myTimeToString(time.Now().UTC())

			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			store, closeDB := NewFileSystemStore(tmpFile, []Article{old})
			defer closeDB()

			want := editedBase
			want.Edited = myTimeToString(time.Now().UTC().Add(time.Second * time.Duration(1)))

			oldID, _ := store.getArticle(validArticleBase.Slug)
			store.editArticle(oldID, want)

			newID, got := store.getArticle(want.Slug)

			if oldID != newID {
				t.Errorf("Article not patched but replaced, oldID: %d, newID: %d", oldID, newID)
			}
			assertArticleWithoutTime(t, got, want)
			if got.Published != old.Published {
				t.Error("published time changes when editing article")
			}
			if !myStringToTime(got.Edited).After(myStringToTime(old.Edited)) {
				t.Error("edited time not updated when editing article")
			}
		})

		t.Run("delete article", func(t *testing.T) {
			articles := MakeBothTypesOfArticle(5)
			toDelete := articles[4]

			tmpFile, cleanTempFile := makeTempFile()
			defer cleanTempFile()

			store, closeDB := NewFileSystemStore(tmpFile, articles)
			defer closeDB()

			id, _ := store.getArticle(toDelete.Slug)
			store.deleteArticle(id)

			_, got := store.getArticle(toDelete.Slug)
			assertArticle(t, got, Article{})

			if len(store.getAll()) != len(articles)-1 {
				t.Error("article not deleted")
			}
		})
	})
}

package main

import "sort"

type InMemStore struct {
	articles []Article
}

func (i *InMemStore) getAll() []Article {
	return i.articles
}

func (i *InMemStore) getPage(page int, category string) []Article {
	var filtered []Article

	for _, a := range i.getAll() {
		if a.Category == category {
			filtered = append(filtered, a)
		}
	}

	sort.Slice(filtered, func(i int, j int) bool {
		return filtered[i].Published.Before(filtered[j].Published)
	})

	p := page
	if page < 1 {
		p = 1
	}
	maxPage := (len(filtered) / perPage)
	if page > maxPage {
		p = maxPage
	}
	endArticle := (p * perPage) - 1
	if len(filtered) < endArticle {
		endArticle = len(filtered)
	}

	articles := filtered[(p-1)*perPage : endArticle+1]
	return articles
}

func (i *InMemStore) getArticle(slug string) Article {
	for _, a := range i.articles {
		if a.Slug == slug {
			return a
		}
	}
	return Article{}
}

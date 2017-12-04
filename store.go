package main

import (
	"math/rand"
	"time"
)

type Store interface {
	getLink(slug string) *Link
	tryClickLink(slug string) *Link
	getAllLinks() []*Link
	create(url string) (*Link, error)
	createCustom(slug, url string) (*Link, error)
	delete(slug string)
	edit(slug, url string)
}

type Link struct {
	Slug    string    `json:"slug"`
	Url     string    `json:"url"`
	Clicks  int       `json:"clicks"`
	Created time.Time `json:"created"`
}

func generateSlug() string {
	// From: http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
	var chars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, Config.slugLength)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}

	return string(s)
}

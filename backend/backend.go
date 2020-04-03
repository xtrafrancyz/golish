package backend

import (
	"math/rand"
	"time"
)

type Backend interface {
	GetLink(slug string) *Link
	TryClickLink(slug string) *Link
	GetAllLinks() []*Link
	Create(url string) (*Link, error)
	CreateCustom(slug, url string) (*Link, error)
	Delete(slug string)
	Edit(slug, url string)
}

type Link struct {
	Slug    string    `json:"slug"`
	Url     string    `json:"url"`
	Clicks  uint64    `json:"clicks"`
	Created time.Time `json:"created"`
}

func (l *Link) String() string {
	return "Link(" + l.Slug + " => " + l.Url + ")"
}

func generateSlug(length int) string {
	// From: http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
	var chars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, length)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}

	return string(s)
}

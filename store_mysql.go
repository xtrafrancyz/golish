package main

import (
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlStore struct {
	cache map[string]*Link
	db    *sql.DB
}

func newMysqlStore(host, user, password, database string) (*MysqlStore, error) {
	db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+")/"+database+"?parseTime=true")
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `links` (" +
		"`slug` varchar(255) COLLATE utf8_unicode_ci NOT NULL," +
		"`url` text COLLATE utf8_unicode_ci NOT NULL," +
		"`clicks` int(11) NOT NULL DEFAULT '0'," +
		"`created` datetime DEFAULT NULL," +
		"PRIMARY KEY (`slug`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;")
	if err != nil {
		return nil, err
	}

	store := &MysqlStore{
		db:    db,
		cache: make(map[string]*Link),
	}
	return store, nil
}

func (s *MysqlStore) getLink(slug string) *Link {
	link, ok := s.cache[slug]
	if ok {
		return link
	}
	link = &Link{Slug: slug}
	err := s.db.QueryRow("SELECT url, clicks, created FROM links WHERE slug = ?", slug).Scan(&link.Url, &link.Clicks, &link.Created)
	if err != nil && err != sql.ErrNoRows {
		log.Println("mysql error: ", err)
	}
	s.cache[slug] = link
	if link.Url != "" {
		return link
	} else {
		return nil
	}
}

func (s *MysqlStore) tryClickLink(slug string) *Link {
	link := s.getLink(slug)
	if link == nil {
		return nil
	}
	s.db.Exec("UPDATE links SET clicks = clicks + 1 WHERE slug = ?", slug)
	link.Clicks += 1
	return link
}

func (s *MysqlStore) getAllLinks() []*Link {
	rows, err := s.db.Query("SELECT slug, url, clicks, created FROM links")
	list := make([]*Link, 0)
	if err != nil {
		log.Println("mysql error: ", err)
		return list
	}
	for rows.Next() {
		link := Link{}
		rows.Scan(&link.Slug, &link.Url, &link.Clicks, &link.Created)
		s.cache[link.Slug] = &link
		list = append(list, &link)
	}
	return list
}

func (s *MysqlStore) create(url string) (*Link, error) {
	var slug string
	err := s.db.QueryRow("SELECT slug FROM links WHERE url = ?", url).Scan(&slug)
	if err == nil {
		return s.getLink(slug), nil
	}
	for {
		slug = generateSlug()
		rows, err := s.db.Query("SELECT 1 FROM links WHERE slug = ?", slug)
		if err != nil {
			log.Println("mysql error: ", err)
			return nil, errors.New("database error")
		}
		// this slug is not used
		if !rows.Next() {
			break
		}
	}
	return s.createCustom(slug, url)
}

func (s *MysqlStore) createCustom(slug, url string) (*Link, error) {
	link := s.getLink(slug)
	if link != nil {
		return nil, errors.New("slug '" + slug + "' is already exists")
	} else {
		return s.createNow(slug, url)
	}
}

func (s *MysqlStore) createNow(slug, url string) (*Link, error) {
	_, err := s.db.Exec("INSERT INTO links (slug, url, created) VALUES (?, ?, ?)", slug, url, time.Now())
	if err != nil {
		log.Println("mysql error: ", err)
		return nil, errors.New("database error")
	}
	link := &Link{
		Slug:    slug,
		Url:     url,
		Created: time.Now(),
	}
	s.cache[slug] = link
	return link, nil
}

func (s *MysqlStore) delete(slug string) {
	delete(s.cache, slug)
	_, err := s.db.Exec("DELETE FROM links WHERE slug = ?", slug)
	if err != nil {
		log.Println("mysql error: ", err)
	}
}

func (s *MysqlStore) edit(slug, url string) {
	log.Print(slug, url)
	delete(s.cache, slug)
	_, err := s.db.Exec("UPDATE links SET url = ? WHERE slug = ?", url, slug)
	if err != nil {
		log.Println("mysql error: ", err)
	}
}

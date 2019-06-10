package backend

import (
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Mysql struct {
	cache      *sync.Map
	db         *sql.DB
	slugLength int
}

type internalLink struct {
	*Link
	lastUsed time.Time
	dirty    bool
}

func (l *internalLink) use() {
	l.lastUsed = time.Now()
}

func NewMysql(slugLength int, host, user, password, database string) (*Mysql, error) {
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

	backend := &Mysql{
		cache:      &sync.Map{},
		db:         db,
		slugLength: slugLength,
	}

	go func() {
		for range time.Tick(1 * time.Minute) {
			deadline := time.Now().Add(-10 * time.Minute)
			backend.cache.Range(func(key, value interface{}) bool {
				link := value.(*internalLink)
				if link.lastUsed.Before(deadline) {
					backend.cache.Delete(key)
				} else if link.dirty {
					_, err := backend.db.Exec("UPDATE links SET clicks = ? WHERE slug = ?", link.Clicks, link.Slug)
					if err != nil {
						log.Println("mysql error: ", err)
					}
					link.dirty = false
				}
				return true
			})
		}
	}()

	return backend, nil
}

func (s *Mysql) getInternalLink(slug string) *internalLink {
	link0, ok := s.cache.Load(slug)
	if ok {
		link := link0.(*internalLink)
		link.use()
		return link
	}
	link := &internalLink{
		Link: &Link{
			Slug: slug,
		},
	}
	link.use()
	err := s.db.QueryRow("SELECT url, clicks, created FROM links WHERE slug = ?", slug).Scan(&link.Url, &link.Clicks, &link.Created)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("mysql error: ", err)
		} else {
			link.Link = nil
		}
	}
	s.cache.Store(slug, link)
	return link
}

func (s *Mysql) GetLink(slug string) *Link {
	return s.getInternalLink(slug).Link
}

func (s *Mysql) TryClickLink(slug string) *Link {
	link := s.getInternalLink(slug)
	if link.Link == nil {
		return nil
	}
	link.Clicks++
	link.dirty = true
	return link.Link
}

func (s *Mysql) GetAllLinks() []*Link {
	rows, err := s.db.Query("SELECT slug, url, clicks, created FROM links")
	list := make([]*Link, 0)
	if err != nil {
		log.Println("mysql error: ", err)
		return list
	}
	for rows.Next() {
		link := &internalLink{
			Link: &Link{},
		}
		link.use()
		err = rows.Scan(&link.Slug, &link.Url, &link.Clicks, &link.Created)
		if err != nil {
			log.Println("mysql error: ", err)
			break
		}
		s.cache.Store(link.Slug, link)
		list = append(list, link.Link)
	}
	return list
}

func (s *Mysql) Create(url string) (*Link, error) {
	var slug string
	err := s.db.QueryRow("SELECT slug FROM links WHERE url = ?", url).Scan(&slug)
	if err == nil {
		return s.GetLink(slug), nil
	}
	for {
		slug = generateSlug(s.slugLength)
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
	return s.CreateCustom(slug, url)
}

func (s *Mysql) CreateCustom(slug, url string) (*Link, error) {
	link := s.GetLink(slug)
	if link != nil {
		return nil, errors.New("slug '" + slug + "' is already exists")
	} else {
		_, err := s.db.Exec("INSERT INTO links (slug, url, created) VALUES (?, ?, ?)", slug, url, time.Now())
		if err != nil {
			log.Println("mysql error: ", err)
			return nil, errors.New("database error")
		}
		link := &internalLink{
			Link: &Link{
				Slug:    slug,
				Url:     url,
				Created: time.Now(),
			},
		}
		link.use()
		s.cache.Store(slug, link)
		return link.Link, nil
	}
}

func (s *Mysql) Delete(slug string) {
	s.cache.Delete(slug)
	_, err := s.db.Exec("DELETE FROM links WHERE slug = ?", slug)
	if err != nil {
		log.Println("mysql error: ", err)
	}
}

func (s *Mysql) Edit(slug, url string) {
	s.cache.Delete(slug)
	_, err := s.db.Exec("UPDATE links SET url = ? WHERE slug = ?", url, slug)
	if err != nil {
		log.Println("mysql error: ", err)
	}
}

package backend

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	pathpkg "path"
	"sync"
	"time"
)

type File struct {
	cache      *sync.Map
	file       string
	dirty      bool
	slugLength int
	autosave   *time.Ticker
}

func NewFile(slugLength int, path string) (*File, error) {
	dir := pathpkg.Dir(path)
	if dir != "." {
		_ = os.Mkdir(pathpkg.Dir(path), 0755)
	}

	backend := &File{
		file:       path,
		slugLength: slugLength,
	}

	if _, err := os.Stat(backend.file); os.IsNotExist(err) {
		backend.dirty = true
		backend.cache = &sync.Map{}
		if err = backend.save(); err != nil {
			return nil, err
		}
	} else {
		if err = backend.load(); err != nil {
			return nil, err
		}
	}

	backend.autosave = time.NewTicker(60 * time.Second)
	go func() {
		for {
			<-backend.autosave.C
			backend.save()
		}
	}()

	return backend, nil
}

func (s *File) GetLink(slug string) *Link {
	link, ok := s.cache.Load(slug)
	if ok {
		return link.(*Link)
	} else {
		return nil
	}
}

func (s *File) TryClickLink(slug string) *Link {
	link := s.GetLink(slug)
	if link == nil {
		return nil
	}
	s.dirty = true
	link.Clicks += 1
	return link
}

func (s *File) GetAllLinks() []*Link {
	list := make([]*Link, 0)
	s.cache.Range(func(_, link interface{}) bool {
		list = append(list, link.(*Link))
		return true
	})
	return list
}

func (s *File) Create(url string) (*Link, error) {
	var link *Link = nil
	s.cache.Range(func(_, link0 interface{}) bool {
		if link0.(*Link).Url == url {
			link = link0.(*Link)
			return false
		}
		return true
	})
	if link != nil {
		return link, nil
	}
	var slug string
	for {
		slug = generateSlug(s.slugLength)
		if s.GetLink(slug) == nil {
			break
		}
	}
	return s.CreateCustom(slug, url)
}

func (s *File) CreateCustom(slug, url string) (*Link, error) {
	if s.GetLink(slug) != nil {
		return nil, errors.New("slug '" + slug + "' is already exists")
	} else {
		link := &Link{
			Slug:    slug,
			Url:     url,
			Created: time.Now(),
		}
		s.cache.Store(slug, link)
		s.dirty = true
		s.save()
		return link, nil
	}
}

func (s *File) Delete(slug string) {
	if s.GetLink(slug) != nil {
		s.cache.Delete(slug)
		s.dirty = true
		s.save()
	}
}

func (s *File) Edit(slug, url string) {
	link := s.GetLink(slug)
	if link != nil && link.Url != url {
		link.Url = url
		s.dirty = true
	}
}

func (s *File) save() error {
	if !s.dirty {
		return nil
	}
	s.dirty = false
	temp := make(map[string]*Link)
	s.cache.Range(func(slug, link interface{}) bool {
		temp[slug.(string)] = link.(*Link)
		return true
	})
	bytes, err := json.Marshal(&temp)
	if err != nil {
		return err
	}
	if err = os.WriteFile(s.file, bytes, 0644); err != nil {
		return err
	}
	return nil
}

func (s *File) load() error {
	bytes, err := os.ReadFile(s.file)
	if err != nil {
		return err
	}
	temp := make(map[string]*Link)
	if err = json.Unmarshal(bytes, &temp); err != nil {
		return err
	}
	log.Printf("Loaded %d links", len(temp))
	s.cache = &sync.Map{}
	for slug, link := range temp {
		s.cache.Store(slug, link)
	}
	return nil
}

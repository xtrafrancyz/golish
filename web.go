package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/GeertJohan/go.rice"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"github.com/xtrafrancyz/golish/backend"
	"regexp"
	"errors"
)

type web struct {
	backend    backend.Backend
	adminFiles *rice.Box
}

func (w *web) run() {
	router := routing.New()
	router.Get("/", w.handleRoot)
	router.Get("/<slug:["+Config.slugCharacters+"]+>", w.handleSlug)
	adminGroup := router.Group("/@"+Config.adminPath, w.addACAO)
	adminGroup.Get("/list", w.handleList)
	adminGroup.Post("/create", w.handleCreate)
	adminGroup.Post("/delete", w.handleDelete)
	adminGroup.Post("/edit", w.handleEdit)
	adminGroup.Get("/<file:.*>", w.handleAdminRoot)

	server := &fasthttp.Server{
		Handler:           router.HandleRequest,
		ReduceMemoryUsage: true,
	}

	log.Printf("Starting server on http://%s:%d", Config.host, Config.port)
	err := server.ListenAndServe(fmt.Sprintf("%s:%d", Config.host, Config.port))

	if err != nil {
		log.Fatalf("error in fasthttp server: %s", err)
	}
}

func (w *web) addACAO(c *routing.Context) error {
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")
	return nil
}

func (w *web) handleRoot(c *routing.Context) error {
	if Config.defaultRedirect != "" {
		c.Redirect(Config.defaultRedirect, fasthttp.StatusFound)
	} else {
		c.NotFound()
	}
	return nil
}

func (w *web) handleSlug(c *routing.Context) error {
	log.Print(string(c.URI().Path()))
	full := w.backend.TryClickLink(c.Param("slug"))

	if full != nil {
		c.Redirect(full.Url, fasthttp.StatusFound)
	} else {
		c.NotFound()
	}

	return nil
}

func (w *web) handleAdminRoot(c *routing.Context) error {
	path := "/" + c.Param("file")
	if path == "/" {
		path = "/index.html"
	}
	log.Print("{admin}" + path)
	bytes, err := w.adminFiles.Bytes(path)
	if err != nil {
		c.NotFound()
	} else {
		if strings.HasSuffix(path, ".html") {
			c.Response.Header.Set("Content-Type", "text/html")
		} else if strings.HasSuffix(path, ".css") {
			c.Response.Header.Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".js") {
			c.Response.Header.Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".png") {
			c.Response.Header.Set("Content-Type", "image/png")
		}
		c.SetStatusCode(fasthttp.StatusOK)
		c.Write(bytes)
	}
	return nil
}

func (w *web) handleList(c *routing.Context) error {
	log.Print("{admin}/list")
	links := w.backend.GetAllLinks()
	marshaled, _ := json.Marshal(links)
	c.Response.Header.Set("Content-Type", "application/json")
	c.Write(marshaled)
	return nil
}

func (w *web) handleCreate(c *routing.Context) error {
	url := c.PostArgs().Peek("url")
	slug := c.PostArgs().Peek("slug")
	log.Printf("{admin}/create (url=%s, slug=%s)", url, slug)
	if len(url) == 0 {
		c.SetStatusCode(fasthttp.StatusBadRequest)
	} else {
		var link *backend.Link = nil
		var err error = nil
		if len(slug) == 0 {
			link, err = w.backend.Create(string(url))
		} else {
			if ok, _ := regexp.Match("^["+Config.slugCharacters+"]+$", slug); !ok {
				err = errors.New("slug contains illegal characters")
			} else {
				link, err = w.backend.CreateCustom(string(slug), string(url))
			}
		}
		var marshaled []byte
		if err != nil {
			marshaled, _ = json.Marshal(OperationError{
				Error:   true,
				Message: err.Error(),
			})
		} else {
			marshaled, _ = json.Marshal(link)
		}
		c.Response.Header.Set("Content-Type", "application/json")
		c.Write(marshaled)
	}
	return nil
}

func (w *web) handleDelete(c *routing.Context) error {
	slug := c.PostArgs().Peek("slug")
	log.Printf("{admin}/delete (slug=%s)", slug)
	w.backend.Delete(string(slug))
	c.SetStatusCode(fasthttp.StatusOK)
	return nil
}

func (w *web) handleEdit(c *routing.Context) error {
	slug := c.PostArgs().Peek("slug")
	url := c.PostArgs().Peek("url")
	log.Printf("{admin}/edit (slug=%s, url=%s)", slug, url)
	w.backend.Edit(string(slug), string(url))
	c.SetStatusCode(fasthttp.StatusOK)
	return nil
}

type OperationError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

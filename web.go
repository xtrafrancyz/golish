package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/GeertJohan/go.rice"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"github.com/xtrafrancyz/golish/backend"
)

type web struct {
	backend    backend.Backend
	adminFiles *rice.Box
}

func (w *web) run() {
	r := router.New()
	r.GET("/", w.handleRoot)
	r.GET("/{slug:["+Config.slugCharacters+"]+}", w.handleSlug)
	adminGroup := r.Group("/@" + Config.adminPath)
	adminGroup.GET("/list", w.addACAO(w.handleList))
	adminGroup.POST("/create", w.addACAO(w.handleCreate))
	adminGroup.POST("/delete", w.addACAO(w.handleDelete))
	adminGroup.POST("/edit", w.addACAO(w.handleEdit))
	adminGroup.GET("/{file:*}", w.addACAO(w.handleAdminRoot))

	server := &fasthttp.Server{
		Handler:           r.Handler,
		ReduceMemoryUsage: true,
	}

	log.Printf("Starting server on http://%s:%d", Config.host, Config.port)
	err := server.ListenAndServe(fmt.Sprintf("%s:%d", Config.host, Config.port))

	if err != nil {
		log.Fatalf("error in fasthttp server: %s", err)
	}
}

func (w *web) addACAO(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		h(ctx)
	}
}

func (w *web) handleRoot(ctx *fasthttp.RequestCtx) {
	if Config.defaultRedirect != "" {
		ctx.Redirect(Config.defaultRedirect, fasthttp.StatusFound)
	} else {
		ctx.NotFound()
	}
}

func (w *web) handleSlug(ctx *fasthttp.RequestCtx) {
	log.Print(string(ctx.URI().Path()))
	full := w.backend.TryClickLink(ctx.UserValue("slug").(string))

	if full != nil {
		ctx.Redirect(full.Url, fasthttp.StatusFound)
	} else {
		ctx.NotFound()
	}
}

func (w *web) handleAdminRoot(ctx *fasthttp.RequestCtx) {
	path := "/" + ctx.UserValue("file").(string)
	if path == "//" {
		path = "/index.html"
	}
	log.Print("{admin}" + path)
	bytes, err := w.adminFiles.Bytes(path)
	if err != nil {
		ctx.NotFound()
	} else {
		if strings.HasSuffix(path, ".html") {
			ctx.Response.Header.Set("Content-Type", "text/html")
		} else if strings.HasSuffix(path, ".css") {
			ctx.Response.Header.Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".js") {
			ctx.Response.Header.Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".png") {
			ctx.Response.Header.Set("Content-Type", "image/png")
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
		_, _ = ctx.Write(bytes)
	}
}

func (w *web) handleList(ctx *fasthttp.RequestCtx) {
	log.Print("{admin}/list")
	links := w.backend.GetAllLinks()
	marshaled, _ := json.Marshal(links)
	ctx.Response.Header.Set("Content-Type", "application/json")
	_, _ = ctx.Write(marshaled)
}

func (w *web) handleCreate(ctx *fasthttp.RequestCtx) {
	url := ctx.PostArgs().Peek("url")
	slug := ctx.PostArgs().Peek("slug")
	log.Printf("{admin}/create (url=%s, slug=%s)", url, slug)
	if len(url) == 0 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
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
		ctx.Response.Header.Set("Content-Type", "application/json")
		_, _ = ctx.Write(marshaled)
	}
}

func (w *web) handleDelete(ctx *fasthttp.RequestCtx) {
	slug := ctx.PostArgs().Peek("slug")
	log.Printf("{admin}/delete (slug=%s)", slug)
	w.backend.Delete(string(slug))
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (w *web) handleEdit(ctx *fasthttp.RequestCtx) {
	slug := ctx.PostArgs().Peek("slug")
	url := ctx.PostArgs().Peek("url")
	log.Printf("{admin}/edit (slug=%s, url=%s)", slug, url)
	w.backend.Edit(string(slug), string(url))
	ctx.SetStatusCode(fasthttp.StatusOK)
}

type OperationError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

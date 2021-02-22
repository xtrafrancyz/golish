package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/xtrafrancyz/golish/backend"
)

//go:embed admin/*
var adminFiles embed.FS

type web struct {
	backend backend.Backend
}

func (w *web) run() {
	r := router.New()
	r.GET("/", w.handleRoot)
	r.GET("/{slug:["+Config.slugCharacters+"]+}", w.handleSlug)

	adminGroup := r.Group("/@" + Config.adminPath)
	adminGroup.GET("/list", w.handleList)
	adminGroup.POST("/create", w.handleCreate)
	adminGroup.POST("/delete", w.handleDelete)
	adminGroup.POST("/edit", w.handleEdit)
	files, _ := fs.Sub(adminFiles, "admin")
	adminGroup.GET("/{file:*}", fasthttpadaptor.NewFastHTTPHandler(
		http.StripPrefix("/@"+Config.adminPath, http.FileServer(http.FS(files))),
	))

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

func (w *web) handleRoot(ctx *fasthttp.RequestCtx) {
	if Config.defaultRedirect != "" {
		ctx.Redirect(Config.defaultRedirect, fasthttp.StatusFound)
	} else {
		ctx.NotFound()
	}
}

func (w *web) handleSlug(ctx *fasthttp.RequestCtx) {
	log.Print(string(ctx.URI().Path()))

	// Make a copy of a slug. fasthttp/router's UserValues become invalid after the request is completed.
	slug := copyString(ctx.UserValue("slug").(string))

	full := w.backend.TryClickLink(slug)

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
	bytes, err := adminFiles.ReadFile(path)
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

func copyString(in string) string {
	var b strings.Builder
	b.Grow(len(in))
	b.WriteString(in)
	return b.String()
}

type OperationError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

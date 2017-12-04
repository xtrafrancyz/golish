package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"github.com/vharitonsky/iniflags"
)

var Config struct {
	host            string
	port            int
	slugLength      int
	adminPath       string
	defaultRedirect string

	mysqlHost     string
	mysqlDatabase string
	mysqlUser     string
	mysqlPassword string
}

var store Store
var adminFiles *rice.Box

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&Config.host, "host", "0.0.0.0", "Server host")
	flag.IntVar(&Config.port, "port", 34532, "Server port")
	flag.IntVar(&Config.slugLength, "slug-length", 5, "Length of generated url")
	flag.StringVar(&Config.adminPath, "admin-path", "admin", "Admin path")
	flag.StringVar(&Config.defaultRedirect, "default-redirect", "", "Default redirect path")
	flag.StringVar(&Config.mysqlHost, "mysql-host", "127.0.0.1:3306", "MySQL host")
	flag.StringVar(&Config.mysqlDatabase, "mysql-database", "golish", "MySQL database")
	flag.StringVar(&Config.mysqlUser, "mysql-user", "golish", "MySQL user")
	flag.StringVar(&Config.mysqlPassword, "mysql-password", "golish", "MySQL user")

	iniflags.Parse()

	var err error
	store, err = newMysqlStore(Config.mysqlHost, Config.mysqlUser, Config.mysqlPassword, Config.mysqlDatabase)
	if err != nil {
		log.Fatalf("mysql connection: %s", err)
		return
	}

	adminFiles = rice.MustFindBox("admin")
	adminFiles.HTTPBox()

	router := routing.New()
	router.Get("/", handleRoot)
	router.Get("/<slug:[a-zA-Z0-9]+>", handleSlug)
	adminGroup := router.Group("/@"+Config.adminPath, addACAO)
	adminGroup.Get("/list", handleList)
	adminGroup.Post("/create", handleCreate)
	adminGroup.Post("/delete", handleDelete)
	adminGroup.Post("/edit", handleEdit)
	adminGroup.Get("/<file:.*>", handleAdminRoot)

	server := &fasthttp.Server{
		Handler:           router.HandleRequest,
		ReduceMemoryUsage: true,
	}

	log.Printf("Starting server on http://%s:%d", Config.host, Config.port)
	err = server.ListenAndServe(fmt.Sprintf("%s:%d", Config.host, Config.port))

	if err != nil {
		log.Fatalf("error in fasthttp server: %s", err)
	}
}

func addACAO(c *routing.Context) error {
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")
	return nil
}

func handleRoot(c *routing.Context) error {
	if Config.defaultRedirect != "" {
		c.Redirect(Config.defaultRedirect, fasthttp.StatusFound)
	} else {
		c.NotFound()
	}
	return nil
}

func handleSlug(c *routing.Context) error {
	log.Print(string(c.URI().Path()))
	full := store.tryClickLink(c.Param("slug"))

	if full != nil {
		c.Redirect(full.Url, fasthttp.StatusFound)
	} else {
		c.NotFound()
	}

	return nil
}

func handleAdminRoot(c *routing.Context) error {
	path := "/" + c.Param("file")
	if path == "/" {
		path = "/index.html"
	}
	bytes, err := adminFiles.Bytes(path)
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

func handleList(c *routing.Context) error {
	links := store.getAllLinks()
	marshaled, _ := json.Marshal(links)
	c.Response.Header.Set("Content-Type", "application/json")
	c.Write(marshaled)
	return nil
}

func handleCreate(c *routing.Context) error {
	url := c.PostArgs().Peek("url")
	if len(url) == 0 {
		c.SetStatusCode(fasthttp.StatusBadRequest)
	} else {
		var link *Link
		var err error
		slug := c.PostArgs().Peek("slug")
		if len(slug) == 0 {
			link, err = store.create(string(url))
		} else {
			link, err = store.createCustom(string(slug), string(url))
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

func handleDelete(c *routing.Context) error {
	slug := c.PostArgs().Peek("slug")
	store.delete(string(slug))
	c.SetStatusCode(fasthttp.StatusOK)
	return nil
}

func handleEdit(c *routing.Context) error {
	slug := c.PostArgs().Peek("slug")
	url := c.PostArgs().Peek("url")
	store.edit(string(slug), string(url))
	c.SetStatusCode(fasthttp.StatusOK)
	return nil
}

type OperationError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

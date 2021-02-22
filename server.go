package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/vharitonsky/iniflags"
	"github.com/xtrafrancyz/golish/backend"
)

var Config struct {
	host            string
	port            int
	slugLength      int
	adminPath       string
	defaultRedirect string

	backend string

	// mysql backend
	mysqlHost     string
	mysqlDatabase string
	mysqlUser     string
	mysqlPassword string

	// file backend
	filePath string

	slugCharacters string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&Config.host, "host", "0.0.0.0", "Server host")
	flag.IntVar(&Config.port, "port", 34532, "Server port")
	flag.IntVar(&Config.slugLength, "slug-length", 5, "Length of generated url")
	flag.StringVar(&Config.adminPath, "admin-path", "admin", "Admin path")
	flag.StringVar(&Config.defaultRedirect, "default-redirect", "", "Default redirect path")
	flag.StringVar(&Config.backend, "backend", "file", "Backend name (mysql or file)")
	flag.StringVar(&Config.mysqlHost, "mysql-host", "127.0.0.1:3306", "MySQL host")
	flag.StringVar(&Config.mysqlDatabase, "mysql-database", "golish", "MySQL database")
	flag.StringVar(&Config.mysqlUser, "mysql-user", "golish", "MySQL user")
	flag.StringVar(&Config.mysqlPassword, "mysql-password", "golish", "MySQL user")
	flag.StringVar(&Config.filePath, "file-path", "db.json", "Database file (for file backend)")
	Config.slugCharacters = "a-zA-Z0-9_-"

	iniflags.Parse()

	var err error = nil
	var storage backend.Backend = nil
	if Config.backend == "mysql" {
		storage, err = backend.NewMysql(Config.slugLength, Config.mysqlHost, Config.mysqlUser, Config.mysqlPassword, Config.mysqlDatabase)
	} else if Config.backend == "file" {
		storage, err = backend.NewFile(Config.slugLength, Config.filePath)
	} else {
		log.Fatalf("backend must be 'mysql' or 'file'")
		return
	}
	if err != nil {
		log.Fatalf("backend creation: %s", err)
		return
	}

	web := &web{
		backend: storage,
	}
	web.run()
}

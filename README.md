# Golish
Easy to use **Go** **Li**nk **Sh**ortener. Golish is not intended for tens of millions of shortened links
 because it tries to cache all links to improve performance.

![Golish screenshot](https://i.imgur.com/DAWpaxy.png "Golish screenshot")

# Run
1. Download latest release
2. Create `config.ini` file ([Configuration](#configuration))
3. Run `golish-**** -config config.ini`

# Configuration
- `host` -- Server bind host (default: 0.0.0.0).
- `port` -- Server bind port (default: 34532). Put 80 here for default http server.
- `slug-length` -- Length of the generated short url (default: 5).
- `admin-path` -- Secret path with admin interface (default: admin (`example.com/@admin/`)).
- `default-redirect` -- Address to redirect from root page (root page - `example.com/`).
- `backend` -- Backend type **mysql** or **file** (default: file).
- `mysql-host` -- Mysql server address (default: 127.0.0.1:3306).
- `mysql-user` -- Mysql user (default: golish).
- `mysql-password` -- Mysql password (default: golish).
- `mysql-database` -- Mysql database name (default: golish).
- `file-path` -- Database file path for **file** backend (default: db.json).

### Example of `config.ini` file
```ini
port = 80
admin-path = RANDOM_STRING
default-redirect = https://google.com
backend = mysql
mysql-user = user99
mysql-password = qwerty
```

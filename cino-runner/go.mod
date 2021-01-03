module github.com/alranel/cino/cino-runner

go 1.14

require (
	github.com/alranel/cino/lib v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.4.2
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/otiai10/copy v1.4.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/thoas/go-funk v0.7.0
	go.bug.st/serial v1.1.1
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	gopkg.in/ini.v1 v1.51.0
)

replace github.com/alranel/cino/lib => ../lib

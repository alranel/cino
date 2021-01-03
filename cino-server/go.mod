module github.com/alranel/cino/cino-server

go 1.14

require (
	github.com/alranel/cino/lib v0.0.0-00010101000000-000000000000
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-github/v33 v33.0.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lib/pq v1.0.0
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/thoas/go-funk v0.7.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/text v0.3.4 // indirect
	gopkg.in/ini.v1 v1.62.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/alranel/cino/lib => ../lib

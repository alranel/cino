package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/alranel/cino/lib"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/spf13/viper"
)

type Runner struct {
	ID string
}

var Config struct {
	WS struct {
		Bind string
	}
	Architectures []string
	Runners       []Runner
	DB            lib.DBConfig
	GitHub        struct {
		AppID          int64 `mapstructure:"app_id"`
		Secret         string
		PrivateKeyFile string `mapstructure:"private_key_file"`
	}
}

func LoadConfig(path string) error {
	viper.SetConfigName("cino-server")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.SetDefault("ws.bind", ":8080")
	viper.SetDefault("db.dsn", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB")))
	viper.SetDefault("github.secret", "")
	viper.SetDefault("github.app_id", "")
	viper.SetDefault("github.private_key_file", "")

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Error loading configuration: %w", err)
	}
	if err := viper.ReadConfig(file); err != nil {
		return fmt.Errorf("Error loading configuration: %w", err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("CINO")
	viper.AutomaticEnv()

	viper.Unmarshal(&Config)

	fmt.Printf("%d runners loaded\n", len(Config.Runners))

	return nil
}

func GitHubClient(installationID int64) *github.Client {
	// Shared transport to reuse TCP connections.
	tr := http.DefaultTransport

	itr, err := ghinstallation.NewKeyFromFile(tr, Config.GitHub.AppID, installationID, Config.GitHub.PrivateKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	return github.NewClient(&http.Client{Transport: itr})
}

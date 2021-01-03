package runner

import (
	"os"
	"strings"

	"github.com/alranel/cino/lib"
	"github.com/spf13/viper"
)

type Device struct {
	FQBN     string
	Port     string
	Features []string
}

var Config struct {
	RunnerID string `mapstructure:"runner_id"`
	Wiring   []string
	Devices  []Device
	DB       lib.DBConfig
}

func LoadConfig(path string) error {
	viper.SetConfigName("cino-runner")
	viper.SetConfigType("yaml")

	viper.SetDefault("db.dsn", "localhost:5433")

	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := viper.ReadConfig(file); err != nil {
			return err
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("CINO")
	viper.AutomaticEnv()

	viper.Unmarshal(&Config)

	return nil
}

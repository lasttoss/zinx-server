package configs

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type ConfigMap struct {
	Redis struct {
		Url string `json:"url"`
	}
	Jwt struct {
		Secret string `json:"secret"`
	}
	Database struct {
		Url  string `json:"url"`
		Name string `json:"name"`
	}
	Google struct {
		ClientId string `mapstructure:"client_id"`
	}
	Apple struct {
		ClientId       string `mapstructure:"client_id"`
		GoogleClientId string `mapstructure:"google_client_id"`
	}
}

var ServerConfig ConfigMap

func InitConfig() {
	_ = godotenv.Load(".env")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	viper.SetEnvPrefix("MYAPP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var config ConfigMap
	err := mapstructure.Decode(viper.AllSettings(), &config)
	if err != nil {
		log.Fatalf("Error decoding config, %s", err)
	}
	ServerConfig = config
	log.Printf("Config: %+v", config)
}

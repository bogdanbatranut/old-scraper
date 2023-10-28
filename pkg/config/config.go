package config

import (
	"log"

	"github.com/spf13/viper"
)

// Available config variables
const (
	DBUsername             = "db.username"
	DBPass                 = "db.pass"
	DBHost                 = "db.host"
	DBName                 = "db.name"
	HTTPPort               = "http.port"
	NTFYHost               = "ntfy.host"
	NTFYPort               = "ntfy.port"
	NTFYServiceStatusTopic = "ntfy.topic.serviceStatus"
	NTFYCarsTopic          = "ntfy.topic.cars"
	AppURL                 = "app.url"
)

// Config contains and provides the configuration that is required at runtime
type Config interface {
	GetString(string) string
	GetInt(string) int
	GetInt64(string) int64
	GetBool(string) bool
}

// GetConfig returns the configuration
func GetConfig(path string) (Config, error) {

	// defining that we want to read config from the file named "app" in the provided directory
	viper.SetConfigName("app")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")

	_ = viper.BindEnv(DBUsername, "DB_USERNAME")
	_ = viper.BindEnv(DBPass, "DB_PASS")
	_ = viper.BindEnv(DBHost, "DB_HOST")
	_ = viper.BindEnv(DBName, "DB_NAME")
	_ = viper.BindEnv(HTTPPort, "HTTP_PORT")
	_ = viper.BindEnv(NTFYHost, "NTFY_HOST")
	_ = viper.BindEnv(NTFYPort, "NTFY_PORT")
	_ = viper.BindEnv(NTFYServiceStatusTopic, "NTFY_SERVICE_STATUS_TOPIC")
	_ = viper.BindEnv(AppURL, "APP_URL")

	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	configFileUsed := viper.ConfigFileUsed()
	if len(configFileUsed) == 0 {
		log.Println("no configuration file found")
	} else {
		log.Println("configuration file used")
	}
	return viper.GetViper(), nil
}

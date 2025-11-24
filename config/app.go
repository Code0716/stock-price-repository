package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type App struct {
	AppEnv      string `envconfig:"app_env" default:"local"`
	AppLogLevel string `envconfig:"app_log_level" default:"debug"`
	AppTimezone string `envconfig:"app_timezone" default:"Asia/Tokyo"`
}

var app App

func LoadConfigApp() {
	prefix := ""
	err := envconfig.Process(prefix, &app)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetApp() *App {
	return &app
}

package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

const defaultDotEnvFilePath = ".env"

func LoadEnvConfig() {
	// localからだったら、.envを読み込む
	path := defaultDotEnvFilePath
	if os.Getenv("DOT_ENV_FILE_PATH") != "" {
		path = os.Getenv("DOT_ENV_FILE_PATH")
	}

	if err := godotenv.Load(path); err != nil && os.Getenv("APP_ENV") == "local" {
		log.Printf("failed to load .env file: %s", err.Error())
	}

	LoadConfigDatabase()
	LoadConfigApp()
	LoadConfigYahooFinance()
	LoadConfigSlack()
	LoadConfigJQuants()
	LoadConfigFeatureFlag()
	LoadConfigBOX()
}

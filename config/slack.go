package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigSlack struct {
	SlackBotBaseUrl           string `envconfig:"slack_bot_base_url" default:""`
	SlackNotificationBotToken string `envconfig:"slack_notification_bot_token" default:""`
}

var configSlack ConfigSlack

func LoadConfigSlack() {
	prefix := ""
	err := envconfig.Process(prefix, &configSlack)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func Slack() *ConfigSlack {
	return &configSlack
}

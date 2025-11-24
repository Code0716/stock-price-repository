package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Slack struct {
	SlackBotBaseURL           string `envconfig:"slack_bot_base_url" default:""`
	SlackNotificationBotToken string `envconfig:"slack_notification_bot_token" default:""`
}

var slack Slack

func LoadConfigSlack() {
	prefix := ""
	err := envconfig.Process(prefix, &slack)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetSlack() *Slack {
	return &slack
}

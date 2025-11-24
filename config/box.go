package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type BOX struct {
	PrivateKey string `envconfig:"box_private_key" default:""`
}

var box BOX

func LoadConfigBOX() {
	prefix := ""
	err := envconfig.Process(prefix, &box)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetBOX() *BOX {
	return &box
}

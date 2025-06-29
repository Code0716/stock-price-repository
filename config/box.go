package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigBOX struct {
	PrivateKey string `envconfig:"box_private_key" default:""`
}

var configBOX ConfigBOX

func LoadConfigBOX() {
	prefix := ""
	err := envconfig.Process(prefix, &configBOX)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func BOX() *ConfigBOX {
	return &configBOX
}

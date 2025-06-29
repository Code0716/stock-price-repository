package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigJQuants struct {
	JQuantsBaseURLV1   string `envconfig:"j_quants_base_url_v1" default:""`
	JQuantsMailaddress string `envconfig:"j_quants_mailaddress" default:""`
	JQuantsPassword    string `envconfig:"j_quants_password" default:""`
}

var configJQuants ConfigJQuants

func LoadConfigJQuants() {
	prefix := ""
	err := envconfig.Process(prefix, &configJQuants)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func JQuants() *ConfigJQuants {
	return &configJQuants
}

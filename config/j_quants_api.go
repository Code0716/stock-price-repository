package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type JQuants struct {
	JQuantsBaseURLV2       string `envconfig:"j_quants_base_url_v2" default:""`
	JQuantsBaseURLV2APIKey string `envconfig:"j_quants_base_url_v2_api_key" default:""`
}

var jQuants JQuants

func LoadConfigJQuants() {
	prefix := ""
	err := envconfig.Process(prefix, &jQuants)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetJQuants() *JQuants {
	return &jQuants
}

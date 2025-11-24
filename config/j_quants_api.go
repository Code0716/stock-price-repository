package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type JQuants struct {
	JQuantsBaseURLV1   string `envconfig:"j_quants_base_url_v1" default:""`
	JQuantsMailaddress string `envconfig:"j_quants_mailaddress" default:""`
	JQuantsPassword    string `envconfig:"j_quants_password" default:""`
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

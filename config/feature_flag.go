package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigFeatureFlag struct {
	FeatureFlagStartUseingJQuants bool `envconfig:"start_useing_j_quants" default:""`
}

var configFeatureFlag ConfigFeatureFlag

func LoadConfigFeatureFlag() {
	prefix := ""
	err := envconfig.Process(prefix, &configFeatureFlag)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func FeatureFlag() *ConfigFeatureFlag {
	return &configFeatureFlag
}

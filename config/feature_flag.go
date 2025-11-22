package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigFeatureFlag struct {
	// FeatureFlagStartUsingJQuants bool `envconfig:"start_using_j_quants" default:""`
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

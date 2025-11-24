package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type FeatureFlag struct {
	// FeatureFlagStartUsingJQuants bool `envconfig:"start_using_j_quants" default:""`
}

var featureFlag FeatureFlag

func LoadConfigFeatureFlag() {
	prefix := ""
	err := envconfig.Process(prefix, &featureFlag)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetFeatureFlag() *FeatureFlag {
	return &featureFlag
}

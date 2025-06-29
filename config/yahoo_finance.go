package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigYahooFinance struct {
	BaseURL             string `envconfig:"yahoo_finance_api_base_url" default:""`
	YfinancePyBinaryCMD string `envconfig:"yfinance_py_binary_cmd" default:""`
}

var configYahooFinance ConfigYahooFinance

func LoadConfigYahooFinance() {
	prefix := ""
	err := envconfig.Process(prefix, &configYahooFinance)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func YahooFinance() *ConfigYahooFinance {
	return &configYahooFinance
}

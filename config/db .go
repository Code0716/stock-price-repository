package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type ConfigDatabase struct {
	Dialect      string `envconfig:"stock_price_repository_mysql_dialect" default:""`
	Host         string `envconfig:"stock_price_repository_mysql_host" default:""`
	DBName       string `envconfig:"stock_price_repository_mysql_dbname" default:""`
	Passwd       string `envconfig:"stock_price_repository_mysql_password" default:""`
	Port         string `envconfig:"stock_price_repository_mysql_port" default:""`
	User         string `envconfig:"stock_price_repository_mysql_user" default:""`
	Charset      string `envconfig:"stock_price_repository_mysql_charset" default:""`
	Timezone     string `envconfig:"stock_price_repository_mysql_timezone" default:""`
	RootUser     string `envconfig:"stock_price_repository_mysql_root_user" default:""`
	RootPassword string `envconfig:"stock_price_repository_mysql_root_password" default:""`
	ExportPath   string `envconfig:"stock_price_repository_mysql_sql_export" default:""`
}

var configDatabase ConfigDatabase

func LoadConfigDatabase() {
	prefix := ""
	err := envconfig.Process(prefix, &configDatabase)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func Database() *ConfigDatabase {
	return &configDatabase
}

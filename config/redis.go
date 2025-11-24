package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Redis struct {
	RedisUser     string `envconfig:"redis_user" default:""`
	RedisPassword string `envconfig:"redis_password" default:""`
	RedisHost     string `envconfig:"redis_host" default:"localhost:6379"`
	RedisDB       int    `envconfig:"redis_db" default:"0"`
	// go-redisのデフォルトに合わせるなら runtime.GOMAXPROCS(0) * 10 とする
	RedisNumConnsPerCPU int `envconfig:"redis_num_conns_per_cpu" default:"2"`
}

var redis Redis

func LoadConfigRedis() {
	prefix := ""
	err := envconfig.Process(prefix, &redis)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetRedis() *Redis {
	return &redis
}

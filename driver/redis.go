package driver

import (
	"runtime"
	"time"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

func OpenRedis() *redis.Client {
	c := config.Redis()

	numRedisConnPerCPU := c.RedisNumConnsPerCPU
	poolSize := runtime.GOMAXPROCS(0) * numRedisConnPerCPU

	client := redis.NewClient(&redis.Options{
		Addr:     c.RedisHost,
		Username: c.RedisUser,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
		// TODO: parameter tuning
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		// context timeoutが伝搬するようにする
		ContextTimeoutEnabled: true,
		PoolSize:              poolSize,
		MinIdleConns:          poolSize,
		// 古いコネクションが残り続けて不通にならないように定期的に破棄する
		ConnMaxLifetime: time.Minute,
	})
	if err := redisotel.InstrumentTracing(client); err != nil {
		panic(err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		panic(err)
	}
	return client
}

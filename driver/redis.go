//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"runtime"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/config"
)

func OpenRedis() *redis.Client {
	c := config.GetRedis()

	numRedisConnPerCPU := c.RedisNumConnsPerCPU
	poolSize := runtime.GOMAXPROCS(0) * numRedisConnPerCPU
	// 常時 PoolSize 全部のアイドル接続を持つのは過剰。半数を最小確保とする。
	minIdleConns := poolSize / 2
	if minIdleConns < 1 {
		minIdleConns = 1
	}

	client := redis.NewClient(&redis.Options{
		Addr:     c.RedisHost,
		Username: c.RedisUser,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
		// 操作タイムアウト系: ラズパイ + ローカルネットワーク前提
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		// プール枯渇時の取得待ち上限。明示しないと ReadTimeout+1s に依存して挙動が読みにくくなる。
		PoolTimeout: 10 * time.Second,
		// context timeoutが伝搬するようにする
		ContextTimeoutEnabled: true,
		PoolSize:              poolSize,
		MinIdleConns:          minIdleConns,
		MaxRetries:            3, // リトライを有効化
		// 接続再確立コストを抑えつつ、長時間生きた接続が NW 機器側で切られる前に作り直す。
		// ラズパイ常駐 cron 想定で 5 分。
		ConnMaxLifetime: 5 * time.Minute,
		// アイドルのまま放置された接続を自動破棄。MinIdleConns 維持と合わせて
		// 「最小限のホットプール + 余剰は自然減」のバランスにする。
		ConnMaxIdleTime: 5 * time.Minute,
	})
	if err := redisotel.InstrumentTracing(client); err != nil {
		panic(err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		panic(err)
	}
	return client
}

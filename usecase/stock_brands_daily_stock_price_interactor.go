//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/redis/go-redis/v9"
)

type stockBrandsDailyStockPriceInteractorImpl struct {
	tx                                        database.Transaction
	stockBrandRepository                      repositories.StockBrandRepository
	stockBrandsDailyStockPriceRepository      repositories.StockBrandsDailyPriceRepository
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
	stockAPIClient                            gateway.StockAPIClient
	redisClient                               *redis.Client
	slackAPIClient                            gateway.SlackAPIClient
}

type StockBrandsDailyPriceInteractor interface {
	CreateDailyStockPrice(ctx context.Context, now time.Time) error
	CreateHistoricalDailyStockPrices(ctx context.Context, now time.Time) error
}

func NewStockBrandsDailyPriceInteractor(
	tx database.Transaction,
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	stockAPIClient gateway.StockAPIClient,
	redisClient *redis.Client,
	slackAPIClient gateway.SlackAPIClient,
) StockBrandsDailyPriceInteractor {
	return &stockBrandsDailyStockPriceInteractorImpl{
		tx,
		stockBrandRepository,
		stockBrandsDailyStockPriceRepository,
		stockBrandsDailyPriceForAnalyzeRepository,
		stockAPIClient,
		redisClient,
		slackAPIClient,
	}
}

const (
	// createHistoricalDailyStockPrices
	createHistoricalDailyStockPricesLimitAtOnce                                               int           = 4000
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey string        = "create_historical_daily_stock_price_list_toshyo_stock_brands_by_symbol_stock_price_repository_redis_key"
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL time.Duration = 2 * time.Hour
)

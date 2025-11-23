//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/repositories"
)

type stockBrandInteractorImpl struct {
	tx                                        repositories.Transaction
	stockBrandRepository                      repositories.StockBrandRepository
	stockBrandsDailyPriceRepository           repositories.StockBrandsDailyPriceRepository
	analyzeStockBrandPriceHistoryRepository   repositories.AnalyzeStockBrandPriceHistoryRepository
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
	stockAPIClient                            gateway.StockAPIClient
	redisClient                               *redis.Client
}

type StockBrandInteractor interface {
	UpdateStockBrands(ctx context.Context, t time.Time) error
}

func NewStockBrandInteractor(
	tx repositories.Transaction,
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyPriceRepository repositories.StockBrandsDailyPriceRepository,
	analyzeStockBrandPriceHistoryRepository repositories.AnalyzeStockBrandPriceHistoryRepository,
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	stockAPIClient gateway.StockAPIClient,
	redisClient *redis.Client,
) StockBrandInteractor {
	return &stockBrandInteractorImpl{
		tx,
		stockBrandRepository,
		stockBrandsDailyPriceRepository,
		analyzeStockBrandPriceHistoryRepository,
		stockBrandsDailyPriceForAnalyzeRepository,
		stockAPIClient,
		redisClient,
	}
}

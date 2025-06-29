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

type stockBrandInteractorImpl struct {
	tx                              database.Transaction
	stockBrandRepository            repositories.StockBrandRepository
	stockBrandsDailyPriceRepository repositories.StockBrandsDailyPriceRepository
	stockAPIClient                  gateway.StockAPIClient
	redisClient                     *redis.Client
}

type StockBrandInteractor interface {
	UpdateStockBrands(ctx context.Context, t time.Time) error
}

func NewStockBrandInteractor(
	tx database.Transaction,
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyPriceRepository repositories.StockBrandsDailyPriceRepository,
	stockAPIClient gateway.StockAPIClient,
	redisClient *redis.Client,
) StockBrandInteractor {
	return &stockBrandInteractorImpl{
		tx,
		stockBrandRepository,
		stockBrandsDailyPriceRepository,
		stockAPIClient,
		redisClient,
	}
}

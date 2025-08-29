//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"log"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
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

func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo(stockBrand *models.StockBrand, prices []*gateway.StockPrice, now time.Time) []*models.StockBrandDailyPrice {
	if len(prices) == 0 {
		return nil
	}

	log.Printf("newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo: %s(%s)", stockBrand.Name, stockBrand.TickerSymbol)
	result := make([]*models.StockBrandDailyPrice, 0, len(prices))
	for _, v := range prices {
		if v == nil {
			continue
		}
		if v.High.IsZero() && v.Close.IsZero() && v.Low.IsZero() && v.Open.IsZero() {
			continue
		}
		result = append(result, models.NewStockBrandDailyPrice(
			util.GenerateUUID(),
			stockBrand.ID,
			v.Date,
			v.TickerSymbol,
			v.High,
			v.Low,
			v.Open,
			v.Close,
			v.Volume,
			v.AdjustmentClose,
			now,
			now,
		))
	}
	return result

}

// newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice(
	prices []*models.StockBrandDailyPrice,
	now time.Time,
) []*models.StockBrandDailyPriceForAnalyze {
	if len(prices) == 0 {
		return nil
	}

	result := make([]*models.StockBrandDailyPriceForAnalyze, 0, len(prices))
	for _, v := range prices {
		if v == nil {
			continue
		}
		if v.High.IsZero() && v.Close.IsZero() && v.Low.IsZero() && v.Open.IsZero() {
			continue
		}
		result = append(result, models.NewStockBrandDailyPriceForAnalyze(
			util.GenerateUUID(),
			v.Date,
			v.TickerSymbol,
			v.High,
			v.Low,
			v.Open,
			v.Close,
			v.Volume,
			v.Adjclose,
			now,
			now,
		))
	}
	return result
}

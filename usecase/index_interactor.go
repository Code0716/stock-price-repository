//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/redis/go-redis/v9"
)

type indexInteractorImpl struct {
	tx               database.Transaction
	nikkeiRepository repositories.NikkeiRepository
	djiRepository    repositories.DjiRepository
	stockAPIClient   gateway.StockAPIClient
	slackAPIClient   gateway.SlackAPIClient
	redisClient      *redis.Client
}

func NewIndexInteractor(
	tx database.Transaction,
	redisClient *redis.Client,
	nikkeiRepository repositories.NikkeiRepository,
	djiRepository repositories.DjiRepository,
	stockAPIClient gateway.StockAPIClient,
	slackAPIClient gateway.SlackAPIClient,
) IndexInteractor {
	return &indexInteractorImpl{
		tx,
		nikkeiRepository,
		djiRepository,
		stockAPIClient,
		slackAPIClient,
		redisClient,
	}
}

type IndexInteractor interface {
	CreateNikkeiAndDjiHistoricalData(ctx context.Context, t time.Time) error
}

// TODO:これはgatewayに置くべき。
func (ii *indexInteractorImpl) apiResponseToModel(info *gateway.StockChartWithRangeAPIResponseInfo, t time.Time) models.IndexStockAverageDailyPrices {
	var result models.IndexStockAverageDailyPrices
	for _, v := range info.Indicator {
		result = append(result, &models.IndexStockAverageDailyPrice{
			Date:      v.Date,
			High:      v.High,
			Low:       v.Low,
			Open:      v.Open,
			Close:     v.Close,
			Volume:    v.Volume,
			Adjclose:  v.AdjustmentClose,
			CreatedAt: t,
			UpdatedAt: t,
		})
	}
	return result
}

const (
	TradingCalendarOpenRedisKey    string        = "trading_calendar_open_stock_price_repository_redis_key"
	TradingCalendarOpenRedisKeyTTL time.Duration = 30 * 24 * time.Hour
)

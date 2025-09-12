//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/pkg/errors"
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

func (ii *indexInteractorImpl) CreateNikkeiAndDjiHistoricalData(ctx context.Context, t time.Time) error {
	// 日経平均の取得
	nikkeiDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolNikkei,
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange10Y,
	)
	if err != nil {
		return errors.Wrap(err, "CreateNikkeiAndDjiHistoricalData")
	}
	// NYダウ兵器の取得
	djiDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolDji,
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange10Y,
	)
	if err != nil {
		return errors.Wrap(err, "CreateNikkeiAndDjiHistoricalData")
	}

	err = ii.tx.DoInTx(ctx, func(ctx context.Context) error {
		if err := ii.nikkeiRepository.CreateNikkeiStockAverageDailyPrices(ctx, ii.apiResponseToModel(nikkeiDailyPrices, t)); err != nil {
			return errors.Wrap(err, "")
		}
		if err := ii.djiRepository.CreateDjiStockAverageDailyPrices(ctx, ii.apiResponseToModel(djiDailyPrices, t)); err != nil {
			return errors.Wrap(err, "")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package usecase

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type indexInteractorImpl struct {
	tx               repositories.Transaction
	nikkeiRepository repositories.NikkeiRepository
	djiRepository    repositories.DjiRepository
	topixRepository  repositories.TopixRepository
	stockAPIClient   gateway.StockAPIClient
	slackAPIClient   gateway.SlackAPIClient
	redisClient      *redis.Client
}

func NewIndexInteractor(
	tx repositories.Transaction,
	redisClient *redis.Client,
	nikkeiRepository repositories.NikkeiRepository,
	djiRepository repositories.DjiRepository,
	topixRepository repositories.TopixRepository,
	stockAPIClient gateway.StockAPIClient,
	slackAPIClient gateway.SlackAPIClient,
) IndexInteractor {
	return &indexInteractorImpl{
		tx:               tx,
		nikkeiRepository: nikkeiRepository,
		djiRepository:    djiRepository,
		topixRepository:  topixRepository,
		stockAPIClient:   stockAPIClient,
		slackAPIClient:   slackAPIClient,
		redisClient:      redisClient,
	}
}

type IndexInteractor interface {
	CreateNikkeiAndDjiHistoricalData(ctx context.Context, t time.Time) error
}

// TODO:これはgatewayに置くべき。
func (ii *indexInteractorImpl) apiResponseToModel(info *gateway.StockChartWithRangeAPIResponseInfo, t time.Time) models.IndexStockAverageDailyPrices {
	var result models.IndexStockAverageDailyPrices
	for _, v := range info.Indicator {
		// Yahoo は祝日・取得時点未確定の日に null を返し decimal ゼロ値になる。
		// ゼロ円行を保存するとベンチマークリターンが -100% に壊れるためスキップする。
		if v.Close.IsZero() {
			continue
		}
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

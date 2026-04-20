package usecase

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

const (
	createHistoricalDailyStockPricesDateCheckpointRedisKey string        = "create_historical_daily_stock_prices_date_checkpoint"
	createHistoricalDailyStockPricesDateCheckpointRedisTTL time.Duration = 7 * 24 * time.Hour
)

// CreateHistoricalDailyStockPrices - 5年分の全銘柄の日足を日付ループで取得して保存する
func (si *stockBrandsDailyStockPriceInteractorImpl) CreateHistoricalDailyStockPrices(ctx context.Context, now time.Time) error {
	// 全主要市場銘柄を取得してsymbol→IDのマップを作る
	allBrands, err := si.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return errors.Wrap(err, "stockBrandRepository.FindAllMainMarkets error")
	}
	brandIDBySymbol := make(map[string]string, len(allBrands))
	for _, b := range allBrands {
		brandIDBySymbol[b.TickerSymbol] = b.ID
	}

	// Redisからチェックポイント日付を取得（再開用）
	startDate := now.AddDate(-5, 0, 0)
	checkpointStr, err := si.redisClient.Get(ctx, createHistoricalDailyStockPricesDateCheckpointRedisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return errors.Wrap(err, "redisClient.Get error")
	}
	if !errors.Is(err, redis.Nil) {
		checkpointDate, parseErr := time.Parse("2006-01-02", checkpointStr)
		if parseErr == nil {
			startDate = checkpointDate.AddDate(0, 0, 1)
		}
	}

	// 日付ループ（土日はスキップ、祝日はAPIが空を返す）
	for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		prices, err := si.stockAPIClient.GetAllBrandDailyPricesByDate(ctx, d)
		if err != nil {
			log.Printf("GetAllBrandDailyPricesByDate error for %s: %v", d.Format("2006-01-02"), err)
			// チェックポイントを保存して終了（次回再開可能）
			return si.saveHistoricalCheckpoint(ctx, d.AddDate(0, 0, -1))
		}

		if len(prices) == 0 {
			// 祝日など取引なし
			continue
		}

		dailyPrices := si.newStockBrandDailyPricesForDate(brandIDBySymbol, prices, now)
		if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, dailyPrices); err != nil {
			log.Printf("CreateStockBrandDailyPrice error: %v", err)
		}
		if err := si.stockBrandsDailyPriceForAnalyzeRepository.
			CreateStockBrandDailyPriceForAnalyze(
				ctx,
				si.newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice(dailyPrices, now),
			); err != nil {
			log.Printf("CreateStockBrandDailyPriceForAnalyze error: %v", err)
		}

		if err := si.saveHistoricalCheckpoint(ctx, d); err != nil {
			log.Printf("saveHistoricalCheckpoint error: %v", err)
		}
	}

	// 完了したらチェックポイントを削除
	si.redisClient.Del(ctx, createHistoricalDailyStockPricesDateCheckpointRedisKey)
	return nil
}

func (si *stockBrandsDailyStockPriceInteractorImpl) saveHistoricalCheckpoint(ctx context.Context, d time.Time) error {
	return si.redisClient.Set(
		ctx,
		createHistoricalDailyStockPricesDateCheckpointRedisKey,
		d.Format("2006-01-02"),
		createHistoricalDailyStockPricesDateCheckpointRedisTTL,
	).Err()
}

// newStockBrandDailyPricesForDate - 1日分の全銘柄価格をDBモデルに変換する
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPricesForDate(
	brandIDBySymbol map[string]string,
	prices []*gateway.StockPrice,
	now time.Time,
) []*models.StockBrandDailyPrice {
	result := make([]*models.StockBrandDailyPrice, 0, len(prices))
	for _, v := range prices {
		if v.High.IsZero() && v.Close.IsZero() && v.Low.IsZero() && v.Open.IsZero() {
			continue
		}
		stockBrandID, ok := brandIDBySymbol[v.TickerSymbol]
		if !ok {
			continue
		}
		result = append(result, models.NewStockBrandDailyPrice(
			util.GenerateUUID(),
			stockBrandID,
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

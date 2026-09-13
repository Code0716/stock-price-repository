package usecase

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

const (
	rebuildAnalyzeDailyPricesDateCheckpointRedisKey string        = "rebuild_analyze_daily_prices_date_checkpoint"
	rebuildAnalyzeDailyPricesDateCheckpointRedisTTL time.Duration = 30 * 24 * time.Hour
)

// RebuildAnalyzeDailyPrices - stock_brands_daily_price_for_analyze を一度TRUNCATEしたうえで、
// 全主要市場銘柄のJ-Quants調整済みOHLCVを過去5年分取得し直して再構築する。
//
// 既存データが「2023-08〜2025-11の分割が未調整のまま残っている」等の理由で壊れている場合の
// 一度きりの立て直し用（applyCorporateActionsForAnalyzeによる日次の自動遡及補正だけでは、
// 導入前に発生した過去の分割は直せないため）。raw(stock_brands_daily_price)には一切触れない。
//
// 実行前に stock_brands_daily_price_for_analyze / applied_stock_splits_history /
// applied_stock_consolidations_history をTRUNCATEする破壊的操作。CLI側で明示的な
// 確認フラグ(--yes-truncate)を要求すること。
//
// Redisにチェックポイント(日付)を保存し、API障害等で中断しても途中から再開できる
// （create_historical_daily_stock_prices と同じパターン）。
func (si *stockBrandsDailyStockPriceInteractorImpl) RebuildAnalyzeDailyPrices(ctx context.Context, now time.Time) error {
	startDate, err := si.resolveRebuildStartDate(ctx, now)
	if err != nil {
		return errors.Wrap(err, "resolveRebuildStartDate error")
	}

	mainMarketSymbols, err := si.listMainMarketSymbols(ctx)
	if err != nil {
		return errors.Wrap(err, "listMainMarketSymbols error")
	}
	log.Printf("RebuildAnalyzeDailyPrices: rebuilding for %d main market symbols, from %s to %s",
		len(mainMarketSymbols), startDate.Format("2006-01-02"), now.Format("2006-01-02"))

	for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		prices, err := si.stockAPIClient.GetAllBrandDailyPricesByDate(ctx, d)
		if err != nil {
			log.Printf("GetAllBrandDailyPricesByDate error for %s: %v", d.Format("2006-01-02"), err)
			return si.saveRebuildAnalyzeCheckpoint(ctx, d.AddDate(0, 0, -1))
		}
		if len(prices) == 0 {
			// 祝日など取引なし
			continue
		}

		si.rebuildAnalyzePricesForDate(ctx, prices, mainMarketSymbols, d, now)

		if err := si.saveRebuildAnalyzeCheckpoint(ctx, d); err != nil {
			log.Printf("saveRebuildAnalyzeCheckpoint error: %v", err)
		}
	}

	si.redisClient.Del(ctx, rebuildAnalyzeDailyPricesDateCheckpointRedisKey)
	log.Println("RebuildAnalyzeDailyPrices: completed")
	return nil
}

// resolveRebuildStartDate - チェックポイントが無ければTRUNCATEして5年前から、
// あればその翌日から再開する開始日を決める。
func (si *stockBrandsDailyStockPriceInteractorImpl) resolveRebuildStartDate(ctx context.Context, now time.Time) (time.Time, error) {
	startDate := now.AddDate(-analyzeCorporateActionLookbackYears, 0, 0)

	checkpointStr, err := si.redisClient.Get(ctx, rebuildAnalyzeDailyPricesDateCheckpointRedisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return startDate, errors.Wrap(err, "redisClient.Get error")
	}

	if !errors.Is(err, redis.Nil) {
		checkpointDate, parseErr := time.Parse("2006-01-02", checkpointStr)
		if parseErr == nil {
			log.Printf("RebuildAnalyzeDailyPrices: resuming from checkpoint %s", checkpointStr)
			return checkpointDate.AddDate(0, 0, 1), nil
		}
	}

	// チェックポイントが無い＝新規実行。TRUNCATEしてゼロから作り直す。
	log.Println("RebuildAnalyzeDailyPrices: no checkpoint found. truncating stock_brands_daily_price_for_analyze / applied_stock_splits_history / applied_stock_consolidations_history")
	if err := si.stockBrandsDailyPriceForAnalyzeRepository.TruncateAll(ctx); err != nil {
		return startDate, errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.TruncateAll error")
	}
	if err := si.appliedStockSplitsHistoryRepository.TruncateAll(ctx); err != nil {
		return startDate, errors.Wrap(err, "appliedStockSplitsHistoryRepository.TruncateAll error")
	}
	if err := si.appliedStockConsolidationsHistoryRepository.TruncateAll(ctx); err != nil {
		return startDate, errors.Wrap(err, "appliedStockConsolidationsHistoryRepository.TruncateAll error")
	}
	return startDate, nil
}

func (si *stockBrandsDailyStockPriceInteractorImpl) listMainMarketSymbols(ctx context.Context) (map[string]struct{}, error) {
	allBrands, err := si.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "stockBrandRepository.FindAllMainMarkets error")
	}
	mainMarketSymbols := make(map[string]struct{}, len(allBrands))
	for _, b := range allBrands {
		mainMarketSymbols[b.TickerSymbol] = struct{}{}
	}
	return mainMarketSymbols, nil
}

// rebuildAnalyzePricesForDate - 1日分の取得結果を主要市場銘柄に絞ってfor_analyzeへ保存する。
// 1日分の保存失敗でバッチ全体を止めない（ログに残して次の日へ進む）。
func (si *stockBrandsDailyStockPriceInteractorImpl) rebuildAnalyzePricesForDate(
	ctx context.Context,
	prices []*gateway.StockPrice,
	mainMarketSymbols map[string]struct{},
	date time.Time,
	now time.Time,
) {
	mainMarketPrices := filterGatewayPricesBySymbols(prices, mainMarketSymbols)
	if len(mainMarketPrices) == 0 {
		return
	}

	analyzePrices := newStockBrandDailyPriceForAnalyzeFromGatewayPrices(mainMarketPrices, now)
	if err := si.stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze(ctx, analyzePrices); err != nil {
		log.Printf("CreateStockBrandDailyPriceForAnalyze error for %s: %v", date.Format("2006-01-02"), err)
	}
}

// filterGatewayPricesBySymbols allowedSymbolsに含まれる銘柄のみを残す（主要市場以外の除外用）。
func filterGatewayPricesBySymbols(prices []*gateway.StockPrice, allowedSymbols map[string]struct{}) []*gateway.StockPrice {
	result := make([]*gateway.StockPrice, 0, len(prices))
	for _, p := range prices {
		if p == nil {
			continue
		}
		if _, ok := allowedSymbols[p.TickerSymbol]; ok {
			result = append(result, p)
		}
	}
	return result
}

func (si *stockBrandsDailyStockPriceInteractorImpl) saveRebuildAnalyzeCheckpoint(ctx context.Context, d time.Time) error {
	return si.redisClient.Set(
		ctx,
		rebuildAnalyzeDailyPricesDateCheckpointRedisKey,
		d.Format("2006-01-02"),
		rebuildAnalyzeDailyPricesDateCheckpointRedisTTL,
	).Err()
}

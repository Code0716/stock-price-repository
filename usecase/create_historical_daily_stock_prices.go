package usecase

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	// createHistoricalDailyStockPrices
	createHistoricalDailyStockPricesLimitAtOnce                                               int           = 4000
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey string        = "create_historical_daily_stock_price_list_toshyo_stock_brands_by_symbol_stock_price_repository_redis_key"
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL time.Duration = 2 * time.Hour
)

// CreateHistoricalDailyStockPrices - 5年分の全銘柄の日足を取得して保存する
func (si *stockBrandsDailyStockPriceInteractorImpl) CreateHistoricalDailyStockPrices(ctx context.Context, now time.Time) error {
	symbolFrom, err := si.redisClient.Get(
		ctx,
		createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey,
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return errors.Wrap(err, "redisClient.Get error")
	}
	if errors.Is(err, redis.Nil) {
		// なければ最初から
		// ってかもういらないね。一発でいけるんだから。
		symbolFrom = "0"
	}

	// 銘柄取得
	// FindAllで取得してもいいが、API次第もあるので一旦制御
	stockBrands, err := si.stockBrandRepository.
		FindFromSymbol(ctx, symbolFrom, createHistoricalDailyStockPricesLimitAtOnce)
	if err != nil {
		return errors.Wrap(err, "stockBrandRepository.FindAll")
	}

	if err = si.createHistoricalDailyStockPrices(ctx, stockBrands, now); err != nil {
		return errors.Wrap(err, "createHistoricalDailyStockPrices error")
	}

	if len(stockBrands) == 0 {
		return nil
	}

	err = si.redisClient.SetEx(
		ctx,
		createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey,
		stockBrands[len(stockBrands)-1].TickerSymbol,
		createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL,
	).Err()
	if err != nil {
		return errors.Wrap(err, "redisClient.Set error")
	}

	return nil
}

// createHistoricalDailyStockPrices - 5年分の全銘柄の日足を取得して保存する
func (si *stockBrandsDailyStockPriceInteractorImpl) createHistoricalDailyStockPrices(ctx context.Context, stockBrands []*models.StockBrand, now time.Time) error {
	var wg sync.WaitGroup
	numCPU := runtime.NumCPU()
	stockBrandsCh := make(chan *models.StockBrand, len(stockBrands))
	// 数が多くなりすぎるのでnumCPUにしておく。
	stockBrandDailyPricesCh := make(chan []*models.StockBrandDailyPrice, numCPU)
	log.Printf("cpu num : %d", numCPU)

	// workerの起動
	for w := 1; w <= numCPU; w++ {
		wg.Add(1)
		go si.createHistoricalDailyStockPricesBySymbol(ctx, &wg, stockBrandsCh, stockBrandDailyPricesCh, now)
	}

	for _, v := range stockBrands {
		stockBrandsCh <- v
	}
	close(stockBrandsCh)

	go func() {
		wg.Wait()
		close(stockBrandDailyPricesCh)
	}()

	for v := range stockBrandDailyPricesCh {
		if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, v); err != nil {
			log.Printf("stockBrandsDailyStockPriceRepository.CreateStockBrandsDailyPrice error: %v", err)
		}
		if err := si.stockBrandsDailyPriceForAnalyzeRepository.
			CreateStockBrandDailyPriceForAnalyze(
				ctx,
				si.newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice(v, now),
			); err != nil {
			log.Printf("stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandsDailyPriceForAnalyze error: %v", err)
		}
	}

	return nil
}

// createHistoricalDailyStockPricesBySymbol - 銘柄ごとに5年分の日足を証券コードから作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) createHistoricalDailyStockPricesBySymbol(
	ctx context.Context,
	wg *sync.WaitGroup,
	stockBrandsCh <-chan *models.StockBrand,
	stockBrandDailyPricesCh chan<- []*models.StockBrandDailyPrice,
	now time.Time,
) {
	defer wg.Done()
	for v := range stockBrandsCh {
		resp, err := si.stockAPIClient.GetDailyPricesBySymbolAndRange(
			ctx,
			gateway.StockAPISymbol(v.TickerSymbol),
			now.AddDate(-5, 0, 0),
			now,
		)
		if err != nil {
			log.Printf("stockAPIClient.GetDailyPricesBySymbolAndRange error: %v", err)
			continue
		}

		stockBrandDailyPricesCh <- si.newStockBrandDailyPricesByStockPrice(v, resp, now)
	}
}

// newStockBrandDailyPricesByStockPrice - models.StockBrandDailyPriceスライスの作成
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPricesByStockPrice(stockBrand *models.StockBrand, stockPrices []*gateway.StockPrice, now time.Time) []*models.StockBrandDailyPrice {
	if stockPrices == nil {
		return nil
	}

	log.Printf("newStockBrandDailyPricesByStockPrice: %s(%s), %d", stockBrand.Name, stockBrand.TickerSymbol, len(stockPrices))
	result := make([]*models.StockBrandDailyPrice, 0, len(stockPrices))
	for _, v := range stockPrices {
		if v.High.IsZero() && v.Close.IsZero() && v.Low.IsZero() && v.Open.IsZero() {
			continue
		}

		result = append(
			result,
			models.NewStockBrandDailyPrice(
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

// newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo model.StockBrandDailyPriceスライスの作成
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

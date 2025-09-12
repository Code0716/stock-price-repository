//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/pkg/errors"
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
	// createDailyStockPrice
	createDailyStockPriceLimitAtOnceByAll                                          int           = 4000
	createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey string        = "create_daily_stock_price_list_toshyo_stock_brands_by_symbol_stock_price_repository_redis_key"
	createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL time.Duration = 2 * time.Hour

	// createHistoricalDailyStockPrices
	createHistoricalDailyStockPricesLimitAtOnce                                               int           = 4000
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey string        = "create_historical_daily_stock_price_list_toshyo_stock_brands_by_symbol_stock_price_repository_redis_key"
	createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL time.Duration = 2 * time.Hour
)

// CreateDailyStockPrice - 全銘柄の日足を取得して保存する
func (si *stockBrandsDailyStockPriceInteractorImpl) CreateDailyStockPrice(ctx context.Context, now time.Time) error {
	symbolFrom, err := si.redisClient.Get(
		ctx,
		createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey,
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return errors.Wrap(err, "redisClient.Get error")
	}

	if errors.Is(err, redis.Nil) {
		// なければ最初から
		symbolFrom = "0"
	}

	// JQuantsを使用する場合は、一度で全てのデータを取得する
	limit := createDailyStockPriceLimitAtOnceByAll

	stockBrands, err := si.stockBrandRepository.
		FindFromSymbol(ctx, symbolFrom, limit)
	if err != nil {
		return errors.Wrap(err, "stockBrandRepository.FindAll")
	}

	if len(stockBrands) == 0 {
		return nil
	}

	if err = si.createDailyStockPrice(ctx, stockBrands, now); err != nil {
		return errors.Wrap(err, "createDailyStockPrice error")
	}

	err = si.redisClient.SetEx(
		ctx,
		createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey,
		stockBrands[len(stockBrands)-1].TickerSymbol,
		createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisTTL,
	).Err()
	if err != nil {
		return errors.Wrap(err, "redisClient.Set error")
	}

	return nil
}

// createDailyStockPrice - 日足を作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) createDailyStockPrice(ctx context.Context, stockBrands []*models.StockBrand, now time.Time) error {
	// バッチサイズの定義
	const (
		// DBへの一括書き込みサイズ
		batchSize = 100
		// チャネルバッファの最大サイズ
		maxBufferSize = 1000
		// ワーカーあたりの推奨バッファ数
		bufferPerWorker = 10
	)

	start := time.Now()
	defer func() {
		log.Printf("Processing completed in %v", time.Since(start))
	}()

	var wg sync.WaitGroup

	// システムのCPU数を取得
	numCPU := runtime.NumCPU()
	// ワーカー数をCPU数に設定
	numWorkers := numCPU

	// バッファサイズの計算
	// 1. ワーカー数 × ワーカーあたりのバッファ数
	// 2. 最大バッファサイズを超えないように制限
	bufferSize := min(numWorkers*bufferPerWorker, maxBufferSize)

	// 入力チャネルは全データを格納できるサイズに
	stockBrandsCh := make(chan *models.StockBrand, len(stockBrands))
	// 出力チャネルは計算されたバッファサイズに
	stockBrandDailyPricesCh := make(chan []*models.StockBrandDailyPrice, bufferSize)

	log.Printf("cpu num: %d, workers: %d, buffer size: %d, batch size: %d",
		numCPU, numWorkers, bufferSize, batchSize)

	// workerの起動
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go si.createDailyStockPriceBySymbol(ctx, &wg, stockBrandsCh, stockBrandDailyPricesCh, now)
	}

	// job
	for _, v := range stockBrands {
		select {
		case <-ctx.Done():
			close(stockBrandsCh)
			return ctx.Err()
		case stockBrandsCh <- v:
		}
	}
	close(stockBrandsCh)

	go func() {
		wg.Wait()
		close(stockBrandDailyPricesCh)
	}()

	// worker
	var batch []*models.StockBrandDailyPrice
	var batchForAnalyze []*models.StockBrandDailyPriceForAnalyze

	for v := range stockBrandDailyPricesCh {
		// チャネルから受け取ったデータをバッチに追加
		batch = append(batch, v...)
		batchForAnalyze = append(batchForAnalyze, si.newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice(v, now)...)

		// バッチサイズが100に達したら処理を実行
		if len(batch) >= batchSize {
			if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, batch); err != nil {
				return errors.Wrap(err, "stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice error: failed to create batch")
			}

			batch = batch[:0] // バッチをリセット
		}
		if len(batchForAnalyze) >= batchSize {
			if err := si.stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze(ctx, batchForAnalyze); err != nil {
				return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze error: failed to create batch")
			}
			batchForAnalyze = batchForAnalyze[:0] // バッチをリセット
		}
	}

	// 残りのバッチを処理
	if len(batch) > 0 {
		if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, batch); err != nil {
			return errors.Wrap(err, "failed to create final batch")
		}
	}

	return nil
}

// createDailyStockPriceBySymbol - 銘柄ごとに日足を証券コードから作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) createDailyStockPriceBySymbol(
	ctx context.Context,
	wg *sync.WaitGroup,
	stockBrandsCh <-chan *models.StockBrand,
	stockBrandDailyPricesCh chan<- []*models.StockBrandDailyPrice,
	now time.Time,
) {
	defer wg.Done()
	for v := range stockBrandsCh {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := si.stockAPIClient.GetDailyPricesBySymbolAndRange(
				ctx,
				gateway.StockAPISymbol(v.TickerSymbol),
				now.AddDate(0, -1, 0),
				now,
			)

			if err != nil {
				log.Printf("stockAPIClient.GetDailyPricesBySymbolAndRange error: %v", err)
				continue
			}
			if len(resp) == 0 {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case stockBrandDailyPricesCh <- si.newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo(v, resp, now):
			}
		}
	}
}

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

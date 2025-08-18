package usecase

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

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

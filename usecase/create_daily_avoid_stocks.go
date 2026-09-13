//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// dailyAvoidStockWindowDays ボラ計算(252営業日)に必要な、リターンの起点となる1本を含めた取得日数。
const dailyAvoidStockWindowDays = domain_service.DailyAvoidVolatilityWindowDays + 1

type CreateDailyAvoidStocksInteractor interface {
	// CreateDailyAvoidStocks 指定日（asOf が nil なら now を基準にした最新営業日）の引け値時点で、
	// 流動性ユニバース内の直近12ヶ月実現ボラティリティ上位20%を「避けるべき銘柄」として保存する。
	// 既に当日分があれば何もしない（冪等）。force=true で洗い替え。
	// 注意: 本バッチは Slack / notification_history には一切書かない。確認導線は front の /avoid-stocks のみ。
	CreateDailyAvoidStocks(ctx context.Context, now time.Time, asOf *time.Time, concurrency int, force bool) error
}

type createDailyAvoidStocksInteractorImpl struct {
	tx                                   repositories.Transaction
	stockBrandsDailyStockPriceRepository repositories.AdjustedDailyPriceRepository
	stockBrandRepository                 repositories.StockBrandRepository
	dailyAvoidStockRepository            repositories.DailyAvoidStockRepository
}

func NewCreateDailyAvoidStocksInteractor(
	tx repositories.Transaction,
	stockBrandsDailyStockPriceRepository repositories.AdjustedDailyPriceRepository,
	stockBrandRepository repositories.StockBrandRepository,
	dailyAvoidStockRepository repositories.DailyAvoidStockRepository,
) CreateDailyAvoidStocksInteractor {
	return &createDailyAvoidStocksInteractorImpl{
		tx:                                   tx,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		stockBrandRepository:                 stockBrandRepository,
		dailyAvoidStockRepository:            dailyAvoidStockRepository,
	}
}

func (ci *createDailyAvoidStocksInteractorImpl) CreateDailyAvoidStocks(ctx context.Context, now time.Time, asOf *time.Time, concurrency int, force bool) error {
	ref := now
	if asOf != nil {
		ref = *asOf
	}

	dates, err := ci.stockBrandsDailyStockPriceRepository.ListRecentTradingDates(ctx, ref, dailyAvoidStockWindowDays)
	if err != nil {
		return errors.Wrap(err, "ListRecentTradingDates error")
	}
	if len(dates) < domain_service.DailyAvoidMinReturns+1 {
		// データ蓄積がボラ判定に必要な最低日数に満たない間は生成しない。
		return nil
	}
	asOfDate := dates[0]
	from := dates[len(dates)-1]

	if !force {
		exists, err := ci.dailyAvoidStockRepository.ExistsByAsOfDate(ctx, asOfDate)
		if err != nil {
			return errors.Wrap(err, "ExistsByAsOfDate error")
		}
		if exists {
			return nil
		}
	}

	rows, err := ci.screen(ctx, asOfDate, from, concurrency)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	if err := ci.tx.DoInTx(ctx, func(ctx context.Context) error {
		if err := ci.dailyAvoidStockRepository.DeleteByAsOfDate(ctx, asOfDate); err != nil {
			return errors.Wrap(err, "DeleteByAsOfDate error")
		}
		if err := ci.dailyAvoidStockRepository.BulkCreate(ctx, rows); err != nil {
			return errors.Wrap(err, "BulkCreate error")
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	log.Printf("daily avoid stocks: created. asOfDate=%s count=%d", asOfDate.Format("2006-01-02"), len(rows))
	return nil
}

// screen 主要市場銘柄を並列にスクリーニングし、避けるべき銘柄（ボラ上位20%）を返す。
func (ci *createDailyAvoidStocksInteractorImpl) screen(
	ctx context.Context,
	asOfDate, from time.Time,
	concurrency int,
) ([]*models.DailyAvoidStock, error) {
	brands, err := ci.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "FindAllMainMarkets error")
	}

	candidates, err := ci.runScreeningWorkers(ctx, brands, from, asOfDate, concurrency)
	if err != nil {
		return nil, err
	}

	return domain_service.RankAvoidCandidates(candidates, asOfDate, domain_service.DefaultDailyAvoidFlagParams()), nil
}

// runScreeningWorkers 固定 concurrency 個のワーカーで全銘柄を並列に評価する（create_daily_stock_picks と同じ設計）。
// 各銘柄の日足は ListDailyPricesBySymbol で銘柄単位にストリーム取得し、252営業日×全銘柄の一括取得による
// メモリ膨張（約110万行）を避ける。
func (ci *createDailyAvoidStocksInteractorImpl) runScreeningWorkers(
	ctx context.Context,
	brands []*models.StockBrand,
	from, to time.Time,
	concurrency int,
) ([]*domain_service.AvoidCandidate, error) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	asc := models.SortOrderAsc
	filter := domain_service.DefaultDailyPickFilterParams()

	workerResults := make([][]*domain_service.AvoidCandidate, concurrency)
	jobs := make(chan *models.StockBrand)
	g, gctx := errgroup.WithContext(ctx)

	for w := 0; w < concurrency; w++ {
		w := w
		g.Go(func() error {
			var local []*domain_service.AvoidCandidate
			for brand := range jobs {
				prices, err := ci.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(gctx, models.ListDailyPricesBySymbolFilter{
					TickerSymbol: brand.TickerSymbol,
					DateFrom:     &from,
					DateTo:       &to,
					DateOrder:    &asc,
				})
				if err != nil {
					return errors.Wrapf(err, "ListDailyPricesBySymbol error symbol=%s", brand.TickerSymbol)
				}
				if c := domain_service.EvaluateAvoidCandidate(brand, prices, filter); c != nil {
					local = append(local, c)
				}
			}
			workerResults[w] = local
			return nil
		})
	}

	g.Go(func() error {
		defer close(jobs)
		for _, brand := range brands {
			select {
			case jobs <- brand:
			case <-gctx.Done():
				return gctx.Err()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, errors.Wrap(err, "runScreeningWorkers error")
	}

	var candidates []*domain_service.AvoidCandidate
	for _, r := range workerResults {
		candidates = append(candidates, r...)
	}
	return candidates, nil
}

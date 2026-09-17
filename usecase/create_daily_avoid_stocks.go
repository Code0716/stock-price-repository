//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"log"
	"runtime"
	"strings"
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

	// BackfillDailyAvoidStocks 直近 days 営業日ぶんをまとめて判定・保存する（過去分のバックフィル用）。
	// 銘柄ごとの価格取得はバックフィル全体で1回に抑え、そこから対象日ごとに該当ウィンドウを
	// スライスして評価する（CreateDailyAvoidStocks を days 回呼ぶと重複取得になるため専用実装にしている）。
	// 1日分の保存に失敗しても残りの日は続行し、最後に失敗した日付をまとめてエラーにする。
	BackfillDailyAvoidStocks(ctx context.Context, now time.Time, days int, concurrency int, force bool) error
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

	return ci.saveDay(ctx, asOfDate, rows)
}

// saveDay 1日分の避けるべき銘柄を保存する（洗い替え）。存在チェックは呼び出し側の責務
// （CreateDailyAvoidStocks は screen 前にまとめてチェック、BackfillDailyAvoidStocks は日付ごとに個別チェックするため）。
func (ci *createDailyAvoidStocksInteractorImpl) saveDay(ctx context.Context, asOfDate time.Time, rows []*models.DailyAvoidStock) error {
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

// BackfillDailyAvoidStocks 直近 days 営業日ぶんをまとめて判定・保存する。
func (ci *createDailyAvoidStocksInteractorImpl) BackfillDailyAvoidStocks(ctx context.Context, now time.Time, days int, concurrency int, force bool) error {
	if days <= 0 {
		return errors.New("days must be positive")
	}

	// 対象日(最大days件) + ボラ計算に必要な予備分をまとめて1回で取得する。
	dates, err := ci.stockBrandsDailyStockPriceRepository.ListRecentTradingDates(ctx, now, days+dailyAvoidStockWindowDays-1)
	if err != nil {
		return errors.Wrap(err, "ListRecentTradingDates error")
	}
	if len(dates) < domain_service.DailyAvoidMinReturns+1 {
		// データ蓄積がボラ判定に必要な最低日数に満たない間は生成しない。
		return nil
	}

	targetCount := min(days, len(dates))
	targetDates := dates[:targetCount] // 新しい順（バックフィル対象日）
	from := dates[len(dates)-1]        // 全体レンジの開始（最古）
	to := dates[0]                     // 全体レンジの終了（最新）

	brands, err := ci.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return errors.Wrap(err, "FindAllMainMarkets error")
	}

	byDate, err := ci.runBackfillScreeningWorkers(ctx, brands, from, to, targetDates, concurrency)
	if err != nil {
		return err
	}

	var failed []string
	for _, asOfDate := range targetDates {
		rows := domain_service.RankAvoidCandidates(byDate[asOfDate], asOfDate, domain_service.DefaultDailyAvoidFlagParams())
		if len(rows) == 0 {
			continue
		}

		if !force {
			exists, err := ci.dailyAvoidStockRepository.ExistsByAsOfDate(ctx, asOfDate)
			if err != nil {
				log.Printf("daily avoid stocks backfill: failed asOfDate=%s err=%+v", asOfDate.Format("2006-01-02"), err)
				failed = append(failed, asOfDate.Format("2006-01-02"))
				continue
			}
			if exists {
				continue // 既にその日の分があればスキップ（バックフィルの冪等性）
			}
		}

		if err := ci.saveDay(ctx, asOfDate, rows); err != nil {
			log.Printf("daily avoid stocks backfill: failed asOfDate=%s err=%+v", asOfDate.Format("2006-01-02"), err)
			failed = append(failed, asOfDate.Format("2006-01-02"))
		}
	}
	if len(failed) > 0 {
		return errors.Errorf("daily avoid stocks backfill: %d date(s) failed: %s", len(failed), strings.Join(failed, ","))
	}

	log.Printf("daily avoid stocks backfill: finished. targetDates=%d", len(targetDates))
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

// runBackfillScreeningWorkers 固定 concurrency 個のワーカーで全銘柄を並列に評価する。
// runScreeningWorkers と異なり、各銘柄の価格系列は [from, to] の全体レンジで1回だけ取得し、
// targetDates の対象日ごとにその日までの部分列を EvaluateAvoidCandidate に渡す
// （BackfillDailyAvoidStocks を days 回呼ぶと同じ価格を重複取得することになるため、これを避けるのが目的）。
func (ci *createDailyAvoidStocksInteractorImpl) runBackfillScreeningWorkers(
	ctx context.Context,
	brands []*models.StockBrand,
	from, to time.Time,
	targetDates []time.Time,
	concurrency int,
) (map[time.Time][]*domain_service.AvoidCandidate, error) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	asc := models.SortOrderAsc
	filter := domain_service.DefaultDailyPickFilterParams()

	workerResults := make([]map[time.Time][]*domain_service.AvoidCandidate, concurrency)
	jobs := make(chan *models.StockBrand)
	g, gctx := errgroup.WithContext(ctx)

	for w := 0; w < concurrency; w++ {
		w := w
		g.Go(func() error {
			local := make(map[time.Time][]*domain_service.AvoidCandidate)
			for brand := range jobs {
				if err := ci.evaluateBrandAcrossDates(gctx, brand, from, to, asc, targetDates, filter, local); err != nil {
					return err
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
		return nil, errors.Wrap(err, "runBackfillScreeningWorkers error")
	}

	merged := make(map[time.Time][]*domain_service.AvoidCandidate, len(targetDates))
	for _, wr := range workerResults {
		for date, cs := range wr {
			merged[date] = append(merged[date], cs...)
		}
	}
	return merged, nil
}

// evaluateBrandAcrossDates 1銘柄の価格系列を1回だけ取得し、targetDates の各日について
// その日までの部分列を EvaluateAvoidCandidate に渡して評価する。結果は date ごとに local へ追記する。
func (ci *createDailyAvoidStocksInteractorImpl) evaluateBrandAcrossDates(
	ctx context.Context,
	brand *models.StockBrand,
	from, to time.Time,
	dateOrder models.SortOrder,
	targetDates []time.Time,
	filter domain_service.DailyPickFilterParams,
	local map[time.Time][]*domain_service.AvoidCandidate,
) error {
	prices, err := ci.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: brand.TickerSymbol,
		DateFrom:     &from,
		DateTo:       &to,
		DateOrder:    &dateOrder,
	})
	if err != nil {
		return errors.Wrapf(err, "ListDailyPricesBySymbol error symbol=%s", brand.TickerSymbol)
	}
	if len(prices) == 0 {
		return nil
	}

	// 日付文字列 → その日を含む部分列の終端インデックス。対象日ごとの再取得を避けるため、
	// 1回取得した prices をスライスし回すだけで済ませる。
	indexByDate := make(map[string]int, len(prices))
	for i, p := range prices {
		indexByDate[p.Date.Format("2006-01-02")] = i
	}
	for _, targetDate := range targetDates {
		idx, ok := indexByDate[targetDate.Format("2006-01-02")]
		if !ok {
			continue // 新規上場前など、対象日に価格が無い銘柄
		}
		if c := domain_service.EvaluateAvoidCandidate(brand, prices[:idx+1], filter); c != nil {
			local[targetDate] = append(local[targetDate], c)
		}
	}
	return nil
}

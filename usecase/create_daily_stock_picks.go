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
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

const (
	// dailyStockPickWindowDays 戦略・指標のウォームアップに必要な営業日数。
	// MovingAverageCross が i>=75、TriangleFormation が window=60 を要求するため十分な余裕を取る。
	dailyStockPickWindowDays = 120
	// dailyStockPickDefaultTopN 通知する銘柄数の既定値。
	dailyStockPickDefaultTopN = 25
	// dailyStockPickDefaultMaxPerSector 同一33業種からの最大採用数の既定値。
	dailyStockPickDefaultMaxPerSector = 4
)

type CreateDailyStockPicksInteractor interface {
	// CreateDailyStockPicks 最新営業日の引け値でスクリーニングし、上位 topN を保存して Slack に通知する。
	// 既に当日分が作成済みかつ全件通知済みなら何もせず正常終了する（冪等）。
	// 作成済みだが未通知（前回 Slack 失敗）の場合は再通知のみ行う。force=true のときは既存を削除して作り直す。
	// topN<=0 は dailyStockPickDefaultTopN、maxPerSector<0 は dailyStockPickDefaultMaxPerSector、
	// concurrency<=0 は runtime.NumCPU() を使う。
	CreateDailyStockPicks(ctx context.Context, now time.Time, topN, maxPerSector, concurrency int, force bool) error
}

type createDailyStockPicksInteractorImpl struct {
	tx                                   repositories.Transaction
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
	stockBrandRepository                 repositories.StockBrandRepository
	dailyStockPickRepository             repositories.DailyStockPickRepository
	slackAPIClient                       gateway.SlackAPIClient
}

func NewCreateDailyStockPicksInteractor(
	tx repositories.Transaction,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	stockBrandRepository repositories.StockBrandRepository,
	dailyStockPickRepository repositories.DailyStockPickRepository,
	slackAPIClient gateway.SlackAPIClient,
) CreateDailyStockPicksInteractor {
	return &createDailyStockPicksInteractorImpl{
		tx:                                   tx,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		stockBrandRepository:                 stockBrandRepository,
		dailyStockPickRepository:             dailyStockPickRepository,
		slackAPIClient:                       slackAPIClient,
	}
}

func (ci *createDailyStockPicksInteractorImpl) CreateDailyStockPicks(ctx context.Context, now time.Time, topN, maxPerSector, concurrency int, force bool) error {
	if topN <= 0 {
		topN = dailyStockPickDefaultTopN
	}
	if maxPerSector < 0 {
		maxPerSector = dailyStockPickDefaultMaxPerSector
	}

	dates, err := ci.stockBrandsDailyStockPriceRepository.ListRecentTradingDates(ctx, now, dailyStockPickWindowDays)
	if err != nil {
		return errors.Wrap(err, "ListRecentTradingDates error")
	}
	if len(dates) < dailyStockPickWindowDays {
		// データ蓄積がウォームアップ日数に満たない間はスクリーニングしない。
		return nil
	}
	pickDate := dates[0]
	from := dates[len(dates)-1]

	if !force {
		existing, err := ci.dailyStockPickRepository.ListByPickDate(ctx, pickDate)
		if err != nil {
			return errors.Wrap(err, "ListByPickDate error")
		}
		if len(existing) > 0 {
			return ci.notifyIfPending(ctx, pickDate, existing)
		}
	}

	picks, err := ci.screen(ctx, pickDate, from, topN, maxPerSector, concurrency)
	if err != nil {
		return err
	}
	if len(picks) == 0 {
		return nil
	}

	if err := ci.tx.DoInTx(ctx, func(ctx context.Context) error {
		if err := ci.dailyStockPickRepository.DeleteByPickDate(ctx, pickDate); err != nil {
			return errors.Wrap(err, "DeleteByPickDate error")
		}
		if err := ci.dailyStockPickRepository.BulkCreate(ctx, picks); err != nil {
			return errors.Wrap(err, "BulkCreate error")
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	return ci.notify(ctx, pickDate, picks)
}

// notifyIfPending 既存の推奨のうち未通知が1件でもあれば再通知する。全件通知済みなら何もしない。
func (ci *createDailyStockPicksInteractorImpl) notifyIfPending(ctx context.Context, pickDate time.Time, existing []*models.DailyStockPick) error {
	for _, p := range existing {
		if !p.Notified() {
			return ci.notify(ctx, pickDate, existing)
		}
	}
	return nil
}

// screen 主要市場銘柄を並列にスクリーニングし、上位 topN の推奨を返す。
func (ci *createDailyStockPicksInteractorImpl) screen(
	ctx context.Context,
	pickDate, from time.Time,
	topN, maxPerSector, concurrency int,
) ([]*models.DailyStockPick, error) {
	brands, err := ci.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "FindAllMainMarkets error")
	}

	candidates, err := ci.runScreeningWorkers(ctx, brands, from, pickDate, concurrency)
	if err != nil {
		return nil, err
	}

	return domain_service.RankDailyPickCandidates(candidates, pickDate, topN, maxPerSector), nil
}

// runScreeningWorkers 固定 concurrency 個のワーカーで全銘柄を並列に評価する（strategy_ranking_interactor.runWorkers と同じ設計）。
// 各銘柄の日足は ListDailyPricesBySymbol で銘柄単位にストリーム取得し、120営業日×全銘柄の一括取得によるメモリ膨張を避ける。
func (ci *createDailyStockPicksInteractorImpl) runScreeningWorkers(
	ctx context.Context,
	brands []*models.StockBrand,
	from, to time.Time,
	concurrency int,
) ([]*domain_service.DailyPickCandidate, error) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	asc := models.SortOrderAsc
	filter := domain_service.DefaultDailyPickFilterParams()
	weights := domain_service.DefaultDailyPickScoreWeights()

	workerResults := make([][]*domain_service.DailyPickCandidate, concurrency)
	jobs := make(chan *models.StockBrand)
	g, gctx := errgroup.WithContext(ctx)

	for w := 0; w < concurrency; w++ {
		w := w
		g.Go(func() error {
			var local []*domain_service.DailyPickCandidate
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
				if c := domain_service.EvaluateDailyPickCandidate(brand, prices, filter, weights); c != nil {
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

	var candidates []*domain_service.DailyPickCandidate
	for _, r := range workerResults {
		candidates = append(candidates, r...)
	}
	return candidates, nil
}

// notify 推奨銘柄を Slack に通知し、成功したら通知日時を記録する。
func (ci *createDailyStockPicksInteractorImpl) notify(ctx context.Context, pickDate time.Time, picks []*models.DailyStockPick) error {
	title, bodies := domain_service.FormatDailyStockPickMessages(picks, domain_service.DailyPickSlackMaxRunes)
	if title == "" {
		return nil
	}

	var ts *string
	for i := range bodies {
		sentTS, err := ci.slackAPIClient.SendMessageByStrings(ctx, gateway.SlackChannelNameExchangeStockInfo, title, &bodies[i], ts)
		if err != nil {
			return errors.Wrapf(err, "SendMessageByStrings error index=%d", i)
		}
		if ts == nil {
			ts = &sentTS
		}
	}

	if err := ci.dailyStockPickRepository.MarkNotified(ctx, pickDate, time.Now()); err != nil {
		return errors.Wrap(err, "MarkNotified error")
	}

	log.Printf("daily stock picks: notified. pickDate=%s count=%d", pickDate.Format("2006-01-02"), len(picks))
	return nil
}

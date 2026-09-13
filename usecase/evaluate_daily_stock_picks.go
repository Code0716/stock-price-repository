//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// dailyStockPickHorizons 答え合わせで評価する保有日数（営業日）。
var dailyStockPickHorizons = []int{1, 3, 5}

// dailyStockPickEvaluationDeadlineDays この日数を過ぎても5営業日後リターンが埋まらない推奨は void 確定させる
// （上場廃止・長期売買停止で永遠に pending になるのを防ぐ）。
const dailyStockPickEvaluationDeadlineDays = 30

type EvaluateDailyStockPicksInteractor interface {
	// EvaluateDailyStockPicks 未確定の推奨について1/3/5営業日後リターンと勝敗を確定させる。
	// 5営業日後がまだ未到来のものは部分的に埋めるだけで evaluated_at は立てず、次回バッチで再試行する。
	EvaluateDailyStockPicks(ctx context.Context, now time.Time) error
}

type evaluateDailyStockPicksInteractorImpl struct {
	tx                                          repositories.Transaction
	dailyStockPickRepository                    repositories.DailyStockPickRepository
	stockBrandsDailyStockPriceRepository        repositories.AdjustedDailyPriceRepository
	appliedStockSplitsHistoryRepository         repositories.AppliedStockSplitsHistoryRepository
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository
}

func NewEvaluateDailyStockPicksInteractor(
	tx repositories.Transaction,
	dailyStockPickRepository repositories.DailyStockPickRepository,
	stockBrandsDailyStockPriceRepository repositories.AdjustedDailyPriceRepository,
	appliedStockSplitsHistoryRepository repositories.AppliedStockSplitsHistoryRepository,
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository,
) EvaluateDailyStockPicksInteractor {
	return &evaluateDailyStockPicksInteractorImpl{
		tx:                                   tx,
		dailyStockPickRepository:             dailyStockPickRepository,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		appliedStockSplitsHistoryRepository:  appliedStockSplitsHistoryRepository,
		appliedStockConsolidationsHistoryRepository: appliedStockConsolidationsHistoryRepository,
	}
}

func (ei *evaluateDailyStockPicksInteractorImpl) EvaluateDailyStockPicks(ctx context.Context, now time.Time) error {
	// 期限切れ判定の対象に含めるため、期限の2倍余裕を持って遡って取得する。
	onOrAfter := now.AddDate(0, 0, -dailyStockPickEvaluationDeadlineDays*2)
	pending, err := ei.dailyStockPickRepository.ListPendingEvaluation(ctx, onOrAfter)
	if err != nil {
		return errors.Wrap(err, "ListPendingEvaluation error")
	}
	if len(pending) == 0 {
		return nil
	}

	byPickDate := make(map[time.Time][]*models.DailyStockPick)
	for _, p := range pending {
		byPickDate[p.PickDate] = append(byPickDate[p.PickDate], p)
	}

	if err := ei.tx.DoInTx(ctx, func(ctx context.Context) error {
		for pickDate, picks := range byPickDate {
			if err := ei.evaluatePickDate(ctx, pickDate, picks, now); err != nil {
				return errors.Wrapf(err, "evaluatePickDate error pickDate=%s", pickDate.Format("2006-01-02"))
			}
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "DoInTx error")
	}
	return nil
}

func (ei *evaluateDailyStockPicksInteractorImpl) evaluatePickDate(ctx context.Context, pickDate time.Time, picks []*models.DailyStockPick, now time.Time) error {
	symbols := uniqueDailyStockPickSymbols(picks)

	prices, err := ei.stockBrandsDailyStockPriceRepository.ListRangePricesBySymbols(ctx, models.ListRangePricesBySymbolsFilter{
		Symbols:  symbols,
		DateFrom: &pickDate,
		DateTo:   &now,
	})
	if err != nil {
		return errors.Wrap(err, "ListRangePricesBySymbols error")
	}
	// 返り値は ticker_symbol, date 昇順のため、シンボルごとに振り分けるだけで日付昇順が保たれる
	// （ForwardReturns は prices[0] が pickDate のバーであることを前提とする）。
	pricesBySymbol := make(map[string][]*models.StockBrandDailyPrice, len(symbols))
	for _, p := range prices {
		pricesBySymbol[p.TickerSymbol] = append(pricesBySymbol[p.TickerSymbol], p)
	}

	expired := now.After(pickDate.AddDate(0, 0, dailyStockPickEvaluationDeadlineDays))

	updated := make([]*models.DailyStockPick, 0, len(picks))
	for _, p := range picks {
		u, err := ei.evaluateOnePick(ctx, p, pricesBySymbol[p.TickerSymbol], now, expired)
		if err != nil {
			return err
		}
		if u != nil {
			updated = append(updated, u)
		}
	}
	if len(updated) == 0 {
		return nil
	}

	if err := ei.dailyStockPickRepository.UpdateEvaluations(ctx, updated); err != nil {
		return errors.Wrap(err, "UpdateEvaluations error")
	}
	return nil
}

// evaluateOnePick 1件の推奨を評価する。まだ確定できず更新すべき内容が無い場合は nil, nil を返す。
func (ei *evaluateDailyStockPicksInteractorImpl) evaluateOnePick(
	ctx context.Context,
	p *models.DailyStockPick,
	prices []*models.StockBrandDailyPrice,
	now time.Time,
	expired bool,
) (*models.DailyStockPick, error) {
	returns, baseFound := domain_service.ForwardReturns(prices, models.AnalyzeStockBrandPriceHistoryActionBuy, dailyStockPickHorizons)
	if !baseFound {
		if !expired {
			// 売買停止等でまだバーが無い。次回バッチで再試行する。
			return nil, nil
		}
		return markDailyStockPickVoid(p, now), nil
	}

	p.Return1D = returns[1]
	p.Return3D = returns[3]

	if returns[5] == nil {
		if !expired {
			// 5営業日後がまだ未到来。Return1D/3Dだけ部分更新し、EvaluatedAtは立てない。
			return p, nil
		}
		return markDailyStockPickVoid(p, now), nil
	}
	p.Return5D = returns[5]

	corporateAction, err := ei.hasCorporateAction(ctx, p.TickerSymbol, prices)
	if err != nil {
		return nil, err
	}
	if corporateAction {
		return markDailyStockPickVoid(p, now), nil
	}

	outcome := domain_service.JudgeDailyPickOutcome(*p.Return5D)
	p.Outcome = &outcome
	evaluatedAt := now
	p.EvaluatedAt = &evaluatedAt
	return p, nil
}

// hasCorporateAction pickDate 翌日から5営業日後までの間に株式分割・併合が適用されたかを確認する。
// 適用があった場合は ForwardReturns の Adjclose 補正に関わらず公正に判定できないため void 扱いとする。
func (ei *evaluateDailyStockPicksInteractorImpl) hasCorporateAction(ctx context.Context, symbol string, prices []*models.StockBrandDailyPrice) (bool, error) {
	limit := 5
	if len(prices)-1 < limit {
		limit = len(prices) - 1
	}
	for i := 1; i <= limit; i++ {
		date := prices[i].Date
		split, err := ei.appliedStockSplitsHistoryRepository.Exists(ctx, symbol, date)
		if err != nil {
			return false, errors.Wrap(err, "AppliedStockSplitsHistoryRepository.Exists error")
		}
		if split {
			return true, nil
		}
		consolidation, err := ei.appliedStockConsolidationsHistoryRepository.Exists(ctx, symbol, date)
		if err != nil {
			return false, errors.Wrap(err, "AppliedStockConsolidationsHistoryRepository.Exists error")
		}
		if consolidation {
			return true, nil
		}
	}
	return false, nil
}

func markDailyStockPickVoid(p *models.DailyStockPick, now time.Time) *models.DailyStockPick {
	voidOutcome := models.DailyStockPickOutcomeVoid
	p.Outcome = &voidOutcome
	evaluatedAt := now
	p.EvaluatedAt = &evaluatedAt
	return p
}

func uniqueDailyStockPickSymbols(picks []*models.DailyStockPick) []string {
	seen := make(map[string]struct{}, len(picks))
	symbols := make([]string, 0, len(picks))
	for _, p := range picks {
		if _, ok := seen[p.TickerSymbol]; ok {
			continue
		}
		seen[p.TickerSymbol] = struct{}{}
		symbols = append(symbols, p.TickerSymbol)
	}
	return symbols
}

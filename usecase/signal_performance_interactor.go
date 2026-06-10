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

const (
	signalPerformanceMaxDays      = 366
	signalPerformanceDefaultDays  = 90
	signalPerformancePriceLookAhead = 25 // +20 horizon の先読み余裕
)

var signalPerformanceHorizons = []int{5, 10, 20}

type signalPerformanceInteractorImpl struct {
	analyzeRepo repositories.AnalyzeStockBrandPriceHistoryRepository
	priceRepo   repositories.StockBrandsDailyPriceRepository
}

// SignalPerformanceInteractor シグナル精度評価
type SignalPerformanceInteractor interface {
	GetSignalPerformance(ctx context.Context, filter *models.SignalPerformanceFilter) (*models.SignalPerformance, error)
}

// NewSignalPerformanceInteractor コンストラクタ
func NewSignalPerformanceInteractor(
	analyzeRepo repositories.AnalyzeStockBrandPriceHistoryRepository,
	priceRepo repositories.StockBrandsDailyPriceRepository,
) SignalPerformanceInteractor {
	return &signalPerformanceInteractorImpl{
		analyzeRepo: analyzeRepo,
		priceRepo:   priceRepo,
	}
}

// GetSignalPerformance 期間内のシグナルを評価し手法別サマリ（+ method 指定時は明細）を返す。
func (s *signalPerformanceInteractorImpl) GetSignalPerformance(ctx context.Context, filter *models.SignalPerformanceFilter) (*models.SignalPerformance, error) {
	signals, err := s.analyzeRepo.FindByCreatedAtRange(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "signalPerformanceInteractorImpl.GetSignalPerformance: FindByCreatedAtRange")
	}

	if len(signals) == 0 {
		return &models.SignalPerformance{
			From:      filter.From,
			To:        filter.To,
			Horizons:  signalPerformanceHorizons,
			Summaries: []*models.SignalPerformanceSummary{},
			Signals:   []*models.EvaluatedSignal{},
		}, nil
	}

	// distinct symbols
	symbolSet := make(map[string]struct{}, len(signals))
	for _, sg := range signals {
		symbolSet[sg.TickerSymbol] = struct{}{}
	}
	symbols := make([]string, 0, len(symbolSet))
	for sym := range symbolSet {
		symbols = append(symbols, sym)
	}

	// シグナル日から最大 horizon 分先まで価格を一括取得（N+1回避）
	priceTo := filter.To.AddDate(0, 0, signalPerformancePriceLookAhead)
	prices, err := s.priceRepo.ListRangePricesBySymbols(ctx, models.ListRangePricesBySymbolsFilter{
		Symbols:  symbols,
		DateFrom: &filter.From,
		DateTo:   &priceTo,
	})
	if err != nil {
		return nil, errors.Wrap(err, "signalPerformanceInteractorImpl.GetSignalPerformance: ListRangePricesBySymbols")
	}

	// symbol → 日付昇順スライスのマップ
	pricesBySymbol := make(map[string][]*models.StockBrandDailyPrice, len(symbols))
	for _, p := range prices {
		pricesBySymbol[p.TickerSymbol] = append(pricesBySymbol[p.TickerSymbol], p)
	}

	evaluated := make([]*models.EvaluatedSignal, 0, len(signals))
	for _, sg := range signals {
		symPrices := pricesBySymbol[sg.TickerSymbol]
		// シグナル日以降の価格だけを渡す（シグナル日当日 = prices[0] = P0）
		from := sg.CreatedAt
		afterSignal := filterPricesFrom(symPrices, from)

		ev := &models.EvaluatedSignal{
			TickerSymbol: sg.TickerSymbol,
			Name:         sg.Name,
			Method:       sg.Method,
			Date:         sg.CreatedAt,
			Action:       sg.Action,
			Score:        sg.Score,
			SignalRank:   sg.SignalRank,
			Memo:         sg.Memo,
		}

		returns, found := domain_service.ForwardReturns(afterSignal, sg.Action, signalPerformanceHorizons)
		if !found {
			// P0 が存在しない → skip（basePrice ゼロ、Returns nil）
			ev.Returns = nil
		} else {
			ev.BasePrice = afterSignal[0].Adjclose
			ev.Returns = returns
		}
		evaluated = append(evaluated, ev)
	}

	summaries := domain_service.AggregateSignalPerformance(evaluated, signalPerformanceHorizons)

	// method 指定時のみ明細を返す
	var detail []*models.EvaluatedSignal
	if filter.Method != "" {
		detail = evaluated
	} else {
		detail = []*models.EvaluatedSignal{}
	}

	return &models.SignalPerformance{
		From:      filter.From,
		To:        filter.To,
		Horizons:  signalPerformanceHorizons,
		Summaries: summaries,
		Signals:   detail,
	}, nil
}

// filterPricesFrom 昇順の prices から signalDate 以降（当日含む）の行を返す。
func filterPricesFrom(prices []*models.StockBrandDailyPrice, signalDate time.Time) []*models.StockBrandDailyPrice {
	sigDay := signalDate.Truncate(24 * time.Hour)
	for i, p := range prices {
		pDay := p.Date.Truncate(24 * time.Hour)
		if !pDay.Before(sigDay) {
			return prices[i:]
		}
	}
	return nil
}

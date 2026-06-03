//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

// minBacktestDays 全戦略の指標ウォームアップに必要な最低日数の目安。
// これ未満は取引0件の結果（空）を返す。
const minBacktestDays = 80

type backtestInteractorImpl struct {
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
}

type BacktestInteractor interface {
	// GetBacktestComparison 指定銘柄・期間で全戦略をバックテストし、トータルリターン降順で返す。
	GetBacktestComparison(ctx context.Context, symbol string, from, to *time.Time, params models.BacktestParams) (*models.BacktestComparison, error)
}

func NewBacktestInteractor(
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
) BacktestInteractor {
	return &backtestInteractorImpl{
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
	}
}

func (b *backtestInteractorImpl) GetBacktestComparison(ctx context.Context, symbol string, from, to *time.Time, params models.BacktestParams) (*models.BacktestComparison, error) {
	order := models.SortOrderAsc
	prices, err := b.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
		DateOrder:    &order,
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	comparison := &models.BacktestComparison{
		Symbol:      symbol,
		TradingDays: len(prices),
		Params:      params,
		Strategies:  []models.StrategyBacktest{},
	}
	if len(prices) > 0 {
		comparison.From = prices[0].Date.Format(util.DateLayout)
		comparison.To = prices[len(prices)-1].Date.Format(util.DateLayout)
	}

	exitParams := domain_service.ExitParams{
		TakeProfit:  params.TakeProfit,
		StopLoss:    params.StopLoss,
		MaxHoldDays: params.MaxHoldDays,
	}

	for _, strategy := range domain_service.StrategyOrder {
		var result models.BacktestResult
		if len(prices) >= minBacktestDays {
			signals := domain_service.EntrySignalsByStrategy(strategy, prices)
			result = domain_service.RunBacktest(prices, signals, exitParams)
		} else {
			// データ不足時は空結果（取引0件）
			result = models.BacktestResult{Equity: []models.BacktestEquityPoint{}, TradeList: []models.BacktestTrade{}}
		}
		comparison.Strategies = append(comparison.Strategies, models.StrategyBacktest{
			Strategy: strategy,
			Label:    domain_service.StrategyLabels[strategy],
			Result:   result,
		})
	}

	// トータルリターン降順でランキング
	sort.SliceStable(comparison.Strategies, func(i, j int) bool {
		return comparison.Strategies[i].Result.TotalReturn.GreaterThan(comparison.Strategies[j].Result.TotalReturn)
	})

	return comparison, nil
}

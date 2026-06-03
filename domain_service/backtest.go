package domain_service

import (
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
)

// ExitParams バックテストの共通イグジット・約定設定。
type ExitParams struct {
	TakeProfit  decimal.Decimal // 利確率（例: 0.10）
	StopLoss    decimal.Decimal // 損切り率（例: 0.05、正の値で指定）
	MaxHoldDays int             // 最大保有営業日数
}

// RunBacktest entrySignals が true の日の終値で1単位買い、共通ルール
// （利確/損切り/最大保有/データ末尾）で終値手仕舞いするバックテストを実行する。
// 同時に保有できるポジションは1つ。エクイティは初期資金 1.0 起点の倍率。
func RunBacktest(prices []*models.StockBrandDailyPrice, entrySignals []bool, params ExitParams) models.BacktestResult {
	n := len(prices)
	result := models.BacktestResult{
		TotalReturn:  decimal.Zero,
		WinRate:      decimal.Zero,
		ProfitFactor: decimal.Zero,
		MaxDrawdown:  decimal.Zero,
		AvgWin:       decimal.Zero,
		AvgLoss:      decimal.Zero,
		PayoffRatio:  decimal.Zero,
		Equity:       []models.BacktestEquityPoint{},
		TradeList:    []models.BacktestTrade{},
	}
	if n == 0 || len(entrySignals) != n {
		return result
	}

	one := decimal.NewFromInt(1)
	realizedEquity := one // 直近に確定した資産（フラット時の資産）
	inPosition := false
	entryIdx := 0
	var entryPrice, entryEquity decimal.Decimal

	equitySeries := make([]decimal.Decimal, 0, n)

	closeTrade := func(i int, reason string) {
		exitPrice := prices[i].Close
		ret := exitPrice.Div(entryPrice).Sub(one)
		realizedEquity = entryEquity.Mul(exitPrice.Div(entryPrice))
		result.TradeList = append(result.TradeList, models.BacktestTrade{
			EntryDate:  prices[entryIdx].Date.Format(util.DateLayout),
			ExitDate:   prices[i].Date.Format(util.DateLayout),
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			Return:     ret,
			HoldDays:   i - entryIdx,
			Reason:     reason,
		})
		inPosition = false
	}

	for i := 0; i < n; i++ {
		// Step A: イグジット判定（エントリー当日は判定しない）
		if inPosition && i > entryIdx {
			ret := prices[i].Close.Div(entryPrice).Sub(one)
			switch {
			case ret.GreaterThanOrEqual(params.TakeProfit):
				closeTrade(i, "take_profit")
			case ret.LessThanOrEqual(params.StopLoss.Neg()):
				closeTrade(i, "stop_loss")
			case i-entryIdx >= params.MaxHoldDays:
				closeTrade(i, "max_hold")
			}
		}

		// Step B: エントリー（フラット時かつシグナル成立）
		if !inPosition && entrySignals[i] {
			inPosition = true
			entryIdx = i
			entryPrice = prices[i].Close
			entryEquity = realizedEquity
		}

		// Step C: 当日のエクイティをマーク（保有中は時価評価）
		var eq decimal.Decimal
		if inPosition {
			eq = entryEquity.Mul(prices[i].Close.Div(entryPrice))
		} else {
			eq = realizedEquity
		}
		equitySeries = append(equitySeries, eq)
		result.Equity = append(result.Equity, models.BacktestEquityPoint{
			Date:   prices[i].Date.Format(util.DateLayout),
			Equity: eq,
		})
	}

	// データ末尾で保有が残っていれば強制クローズ（最終日の終値）。
	// closeTrade 内で realizedEquity が最終確定値に更新される。
	if inPosition {
		closeTrade(n-1, "end_of_data")
	}

	result.Trades = len(result.TradeList)
	result.TotalReturn = realizedEquity.Sub(one)
	result.MaxDrawdown = MaxDrawdown(equitySeries)
	result.AvgHoldDays = avgHoldDays(result.TradeList)
	fillTradeStats(&result)
	return result
}

// fillTradeStats 勝率・PF・平均損益・ペイオフを集計する。
func fillTradeStats(result *models.BacktestResult) {
	if len(result.TradeList) == 0 {
		return
	}
	wins, losses := 0, 0
	grossProfit, grossLoss := decimal.Zero, decimal.Zero
	sumWin, sumLoss := decimal.Zero, decimal.Zero
	for _, t := range result.TradeList {
		if t.Return.GreaterThan(decimal.Zero) {
			wins++
			grossProfit = grossProfit.Add(t.Return)
			sumWin = sumWin.Add(t.Return)
		} else if t.Return.LessThan(decimal.Zero) {
			losses++
			grossLoss = grossLoss.Add(t.Return.Abs())
			sumLoss = sumLoss.Add(t.Return)
		}
	}

	total := decimal.NewFromInt(int64(len(result.TradeList)))
	result.WinRate = decimal.NewFromInt(int64(wins)).Div(total)

	if !grossLoss.IsZero() {
		result.ProfitFactor = grossProfit.Div(grossLoss)
	}
	if wins > 0 {
		result.AvgWin = sumWin.Div(decimal.NewFromInt(int64(wins)))
	}
	if losses > 0 {
		result.AvgLoss = sumLoss.Div(decimal.NewFromInt(int64(losses)))
	}
	if losses > 0 && !result.AvgLoss.IsZero() {
		result.PayoffRatio = result.AvgWin.Div(result.AvgLoss.Abs())
	}
}

func avgHoldDays(trades []models.BacktestTrade) float64 {
	if len(trades) == 0 {
		return 0
	}
	sum := 0
	for _, t := range trades {
		sum += t.HoldDays
	}
	return float64(sum) / float64(len(trades))
}

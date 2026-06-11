package domain_service

import (
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
)

// ExitParams バックテストの共通イグジット・約定設定。
type ExitParams struct {
	TakeProfit     decimal.Decimal // 利確率（例: 0.10）
	StopLoss       decimal.Decimal // 損切り率（例: 0.05、正の値で指定）
	MaxHoldDays    int             // 最大保有営業日数
	CommissionRate decimal.Decimal // 片道手数料率（例: 0.0005）。ゼロ値 = コストなし（後方互換）
	SlippageRate   decimal.Decimal // 片道スリッページ率（例: 0.001）。ゼロ値 = コストなし
}

// tradeStats 約定確定時にインクリメンタルに加算する集計アキュムレータ。
// TradeList を後から走査せずに WinRate/PF/平均損益を算出するために使う。
type tradeStats struct {
	trades      int
	wins        int
	losses      int
	grossProfit decimal.Decimal
	grossLoss   decimal.Decimal
	sumWin      decimal.Decimal
	sumLoss     decimal.Decimal
	holdDaysSum int
}

// record 1約定の損益率と保有日数を集計に加える。
func (ts *tradeStats) record(ret decimal.Decimal, holdDays int) {
	ts.trades++
	ts.holdDaysSum += holdDays
	if ret.GreaterThan(decimal.Zero) {
		ts.wins++
		ts.grossProfit = ts.grossProfit.Add(ret)
		ts.sumWin = ts.sumWin.Add(ret)
	} else if ret.LessThan(decimal.Zero) {
		ts.losses++
		ts.grossLoss = ts.grossLoss.Add(ret.Abs())
		ts.sumLoss = ts.sumLoss.Add(ret)
	}
}

// RunBacktest entrySignals が true の日の終値で1単位買い、共通ルール
// （利確/損切り/最大保有/データ末尾）で終値手仕舞いするバックテストを実行する。
// 同時に保有できるポジションは1つ。エクイティは初期資金 1.0 起点の倍率。
// Equity / TradeList も構築する（フロント表示・ドリルダウン用）。
func RunBacktest(prices []*models.StockBrandDailyPrice, entrySignals []bool, params ExitParams) models.BacktestResult {
	return runBacktest(prices, entrySignals, params, true)
}

// RunBacktestMetrics RunBacktest と同じ成績指標を返すが、Equity / TradeList は構築しない
// （日次の Date.Format とスライス確保を省く）。全銘柄横断バッチなど、集計のみ必要な用途向け。
func RunBacktestMetrics(prices []*models.StockBrandDailyPrice, entrySignals []bool, params ExitParams) models.BacktestResult {
	return runBacktest(prices, entrySignals, params, false)
}

// exitReason 当日のリターンと保有日数から手仕舞い理由を返す。手仕舞い不要なら ""。
// 判定は生の Close ベース（コスト考慮前）で行う。
// コストを判定に含めると TakeProfit / StopLoss の閾値の意味が変わり挙動が複雑化するため。
func exitReason(ret decimal.Decimal, holdDays int, params ExitParams) string {
	switch {
	case ret.GreaterThanOrEqual(params.TakeProfit):
		return "take_profit"
	case ret.LessThanOrEqual(params.StopLoss.Neg()):
		return "stop_loss"
	case holdDays >= params.MaxHoldDays:
		return "max_hold"
	}
	return ""
}

// effectiveEntryPrice 手数料・スリッページを加味した実効エントリー取得単価を返す。
// ゼロ値の場合は生の Close をそのまま返す（後方互換）。
//
//	実効エントリー価格 = close × (1 + slippage) × (1 + commission)
func effectiveEntryPrice(close, commissionRate, slippageRate decimal.Decimal) decimal.Decimal {
	one := decimal.NewFromInt(1)
	price := close
	if !slippageRate.IsZero() {
		price = price.Mul(one.Add(slippageRate))
	}
	if !commissionRate.IsZero() {
		price = price.Mul(one.Add(commissionRate))
	}
	return price
}

// effectiveExitPrice コスト込みの手取りイグジット価格を返す。
//
//	手取りイグジット = close × (1 − slippage) × (1 − commission)
func effectiveExitPrice(close, commissionRate, slippageRate decimal.Decimal) decimal.Decimal {
	one := decimal.NewFromInt(1)
	price := close
	if !slippageRate.IsZero() {
		price = price.Mul(one.Sub(slippageRate))
	}
	if !commissionRate.IsZero() {
		price = price.Mul(one.Sub(commissionRate))
	}
	return price
}

// effectiveExitReturn コスト込みのトレードリターンを返す。
// ゼロ値の場合は生のリターンをそのまま返す（後方互換）。
//
//	リターン = 手取りイグジット ÷ 実効エントリー価格 − 1
func effectiveExitReturn(exitClose, effectiveEntry, commissionRate, slippageRate decimal.Decimal) decimal.Decimal {
	exitEff := effectiveExitPrice(exitClose, commissionRate, slippageRate)
	return exitEff.Div(effectiveEntry).Sub(decimal.NewFromInt(1))
}

func runBacktest(prices []*models.StockBrandDailyPrice, entrySignals []bool, params ExitParams, collectSeries bool) models.BacktestResult {
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
	// entryPrice: コスト込みの実効取得単価（エクイティ計算・リターン算出に使用）
	// entryClose: エントリー当日の生 Close（イグジット判定用）
	var entryPrice, entryClose, entryEquity decimal.Decimal

	equitySeries := make([]decimal.Decimal, 0, n)
	var stats tradeStats

	closeTrade := func(i int, reason string) {
		exitClose := prices[i].Close
		// 利確/損切りの判定は生 Close ベース済み（exitReason で実施）。
		// 約定リターンとエクイティの計算にのみコストを適用する。
		ret := effectiveExitReturn(exitClose, entryPrice, params.CommissionRate, params.SlippageRate)
		// エクイティ更新: 手取りイグジット ÷ 実効エントリー
		exitEffective := effectiveExitPrice(exitClose, params.CommissionRate, params.SlippageRate)
		realizedEquity = entryEquity.Mul(exitEffective.Div(entryPrice))

		// 約定をインクリメンタルに集計（TradeList 非依存）
		stats.record(ret, i-entryIdx)

		if collectSeries {
			result.TradeList = append(result.TradeList, models.BacktestTrade{
				EntryDate:  prices[entryIdx].Date.Format(util.DateLayout),
				ExitDate:   prices[i].Date.Format(util.DateLayout),
				EntryPrice: entryPrice,
				ExitPrice:  exitClose,
				Return:     ret,
				HoldDays:   i - entryIdx,
				Reason:     reason,
			})
		}
		inPosition = false
	}

	for i := 0; i < n; i++ {
		// Step A: イグジット判定（エントリー当日は判定しない）
		// 判定は生 Close ベース（コスト考慮前）
		if inPosition && i > entryIdx {
			rawRet := prices[i].Close.Div(entryClose).Sub(one)
			if reason := exitReason(rawRet, i-entryIdx, params); reason != "" {
				closeTrade(i, reason)
			}
		}

		// Step B: エントリー（フラット時かつシグナル成立）
		if !inPosition && entrySignals[i] {
			inPosition = true
			entryIdx = i
			entryClose = prices[i].Close
			// 実効エントリー価格（コスト込み取得単価）
			entryPrice = effectiveEntryPrice(entryClose, params.CommissionRate, params.SlippageRate)
			entryEquity = realizedEquity
		}

		// Step C: 当日のエクイティをマーク（保有中は時価評価）
		// 時価評価は「生 Close ÷ 実効エントリー価格」ベースで算出する。
		// これによりコスト込みのエクイティカーブが得られる。
		var eq decimal.Decimal
		if inPosition {
			eq = entryEquity.Mul(prices[i].Close.Div(entryPrice))
		} else {
			eq = realizedEquity
		}
		equitySeries = append(equitySeries, eq)
		if collectSeries {
			result.Equity = append(result.Equity, models.BacktestEquityPoint{
				Date:   prices[i].Date.Format(util.DateLayout),
				Equity: eq,
			})
		}
	}

	// データ末尾で保有が残っていれば強制クローズ（最終日の終値）。
	// closeTrade 内で realizedEquity が最終確定値に更新される。
	if inPosition {
		closeTrade(n-1, "end_of_data")
	}

	result.Trades = stats.trades
	result.TotalReturn = realizedEquity.Sub(one)
	result.MaxDrawdown = MaxDrawdown(equitySeries)
	if stats.trades > 0 {
		result.AvgHoldDays = float64(stats.holdDaysSum) / float64(stats.trades)
	}
	fillTradeStats(&result, &stats)
	return result
}

// fillTradeStats 集計済み tradeStats から勝率・PF・平均損益・ペイオフを算出する。
func fillTradeStats(result *models.BacktestResult, stats *tradeStats) {
	if stats.trades == 0 {
		return
	}
	result.WinRate = decimal.NewFromInt(int64(stats.wins)).Div(decimal.NewFromInt(int64(stats.trades)))

	if !stats.grossLoss.IsZero() {
		result.ProfitFactor = stats.grossProfit.Div(stats.grossLoss)
	}
	if stats.wins > 0 {
		result.AvgWin = stats.sumWin.Div(decimal.NewFromInt(int64(stats.wins)))
	}
	if stats.losses > 0 {
		result.AvgLoss = stats.sumLoss.Div(decimal.NewFromInt(int64(stats.losses)))
	}
	if stats.losses > 0 && !result.AvgLoss.IsZero() {
		result.PayoffRatio = result.AvgWin.Div(result.AvgLoss.Abs())
	}
}

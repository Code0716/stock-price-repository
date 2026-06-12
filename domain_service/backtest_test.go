package domain_service

import (
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func pricesFromCloses(closes ...float64) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, 0, len(closes))
	for i, c := range closes {
		out = append(out, &models.StockBrandDailyPrice{
			Date:  base.AddDate(0, 0, i),
			Close: decimal.NewFromFloat(c),
		})
	}
	return out
}

func boolsAt(n int, trueIdx ...int) []bool {
	s := make([]bool, n)
	for _, i := range trueIdx {
		s[i] = true
	}
	return s
}

func TestRunBacktest(t *testing.T) {
	params := ExitParams{
		TakeProfit:  decimal.NewFromFloat(0.10),
		StopLoss:    decimal.NewFromFloat(0.05),
		MaxHoldDays: 10,
	}

	t.Run("利確で手仕舞い", func(t *testing.T) {
		prices := pricesFromCloses(100, 100, 110)
		res := RunBacktest(prices, boolsAt(3, 1), nil, params)
		assert.Equal(t, 1, res.Trades)
		assert.InDelta(t, 0.10, f64FromDec(res.TotalReturn), 1e-9)
		assert.InDelta(t, 1.0, f64FromDec(res.WinRate), 1e-9)
		assert.Equal(t, "take_profit", res.TradeList[0].Reason)
		assert.Len(t, res.Equity, 3)
	})

	t.Run("損切りで手仕舞い", func(t *testing.T) {
		prices := pricesFromCloses(100, 100, 94)
		res := RunBacktest(prices, boolsAt(3, 1), nil, params)
		assert.Equal(t, 1, res.Trades)
		assert.InDelta(t, -0.06, f64FromDec(res.TotalReturn), 1e-9)
		assert.InDelta(t, 0.0, f64FromDec(res.WinRate), 1e-9)
		assert.Equal(t, "stop_loss", res.TradeList[0].Reason)
		assert.InDelta(t, -0.06, f64FromDec(res.MaxDrawdown), 1e-9)
	})

	t.Run("最大保有日数で手仕舞い", func(t *testing.T) {
		p := ExitParams{TakeProfit: decimal.NewFromFloat(0.5), StopLoss: decimal.NewFromFloat(0.5), MaxHoldDays: 2}
		prices := pricesFromCloses(100, 100, 101, 102, 103)
		res := RunBacktest(prices, boolsAt(5, 1), nil, p)
		assert.Equal(t, 1, res.Trades)
		assert.Equal(t, "max_hold", res.TradeList[0].Reason)
		assert.Equal(t, 2, res.TradeList[0].HoldDays)
		assert.InDelta(t, 0.02, f64FromDec(res.TradeList[0].Return), 1e-9) // 102/100-1
	})

	t.Run("データ末尾で強制クローズ", func(t *testing.T) {
		p := ExitParams{TakeProfit: decimal.NewFromFloat(0.5), StopLoss: decimal.NewFromFloat(0.5), MaxHoldDays: 10}
		prices := pricesFromCloses(100, 100, 101)
		res := RunBacktest(prices, boolsAt(3, 1), nil, p)
		assert.Equal(t, 1, res.Trades)
		assert.Equal(t, "end_of_data", res.TradeList[0].Reason)
		assert.InDelta(t, 0.01, f64FromDec(res.TotalReturn), 1e-9)
	})

	t.Run("シグナルなしは取引0件", func(t *testing.T) {
		prices := pricesFromCloses(100, 101, 102)
		res := RunBacktest(prices, boolsAt(3), nil, params)
		assert.Equal(t, 0, res.Trades)
		assert.True(t, res.TotalReturn.IsZero())
		assert.Len(t, res.Equity, 3)
	})

	t.Run("保有中は新規エントリーしない", func(t *testing.T) {
		// day1でエントリー、day2にもシグナルがあるが保有中なので無視、day3でTP
		prices := pricesFromCloses(100, 100, 105, 110)
		res := RunBacktest(prices, boolsAt(4, 1, 2), nil, params)
		assert.Equal(t, 1, res.Trades) // 2回目のシグナルは無視される
	})

	t.Run("空入力やサイズ不一致は空結果", func(t *testing.T) {
		assert.Equal(t, 0, RunBacktest(nil, nil, nil, params).Trades)
		assert.Equal(t, 0, RunBacktest(pricesFromCloses(100, 101), boolsAt(3), nil, params).Trades)
	})
}

func TestRunBacktestMetrics(t *testing.T) {
	params := ExitParams{
		TakeProfit:  decimal.NewFromFloat(0.10),
		StopLoss:    decimal.NewFromFloat(0.05),
		MaxHoldDays: 10,
	}
	// 複数の約定が出るシナリオ（利確→再エントリー→損切り）
	prices := pricesFromCloses(100, 100, 110, 110, 110, 104)
	signals := boolsAt(6, 1, 3)

	full := RunBacktest(prices, signals, nil, params)
	metrics := RunBacktestMetrics(prices, signals, nil, params)

	// メトリクスは完全一致
	assert.Equal(t, full.Trades, metrics.Trades)
	assert.True(t, full.TotalReturn.Equal(metrics.TotalReturn), "TotalReturn")
	assert.True(t, full.WinRate.Equal(metrics.WinRate), "WinRate")
	assert.True(t, full.ProfitFactor.Equal(metrics.ProfitFactor), "ProfitFactor")
	assert.True(t, full.MaxDrawdown.Equal(metrics.MaxDrawdown), "MaxDrawdown")
	assert.True(t, full.AvgWin.Equal(metrics.AvgWin), "AvgWin")
	assert.True(t, full.AvgLoss.Equal(metrics.AvgLoss), "AvgLoss")
	assert.True(t, full.PayoffRatio.Equal(metrics.PayoffRatio), "PayoffRatio")
	assert.InDelta(t, full.AvgHoldDays, metrics.AvgHoldDays, 1e-9)

	// Metrics 版は Equity / TradeList を構築しない
	assert.Empty(t, metrics.Equity)
	assert.Empty(t, metrics.TradeList)
	// Full 版は構築する
	assert.NotEmpty(t, full.Equity)
	assert.NotEmpty(t, full.TradeList)
	assert.GreaterOrEqual(t, full.Trades, 2)
}

func f64FromDec(d decimal.Decimal) float64 {
	v, _ := d.Float64()
	return v
}

func TestRunBacktest_WithCosts(t *testing.T) {
	t.Run("コストゼロ時は既存結果と完全一致", func(t *testing.T) {
		paramsNoCost := ExitParams{
			TakeProfit:  decimal.NewFromFloat(0.10),
			StopLoss:    decimal.NewFromFloat(0.05),
			MaxHoldDays: 10,
		}
		paramsZeroCost := ExitParams{
			TakeProfit:     decimal.NewFromFloat(0.10),
			StopLoss:       decimal.NewFromFloat(0.05),
			MaxHoldDays:    10,
			CommissionRate: decimal.Zero,
			SlippageRate:   decimal.Zero,
		}
		prices := pricesFromCloses(100, 100, 110, 110, 110, 104)
		signals := boolsAt(6, 1, 3)

		resNoCost := RunBacktest(prices, signals, nil, paramsNoCost)
		resZeroCost := RunBacktest(prices, signals, nil, paramsZeroCost)

		assert.True(t, resNoCost.TotalReturn.Equal(resZeroCost.TotalReturn), "TotalReturn 後方互換")
		assert.True(t, resNoCost.WinRate.Equal(resZeroCost.WinRate), "WinRate 後方互換")
		assert.True(t, resNoCost.ProfitFactor.Equal(resZeroCost.ProfitFactor), "ProfitFactor 後方互換")
		assert.Equal(t, resNoCost.Trades, resZeroCost.Trades, "Trades 後方互換")
	})

	t.Run("コストあり: 1トレードのリターンが手計算と一致", func(t *testing.T) {
		// Close: 100 → 110
		// commission = 0.001, slippage = 0.002
		// 実効エントリー価格 = 100 × 1.002 × 1.001 = 100.3002
		// 手取りイグジット = 110 × 0.998 × 0.999 = 109.6901...
		// リターン = 109.6901... / 100.3002 - 1
		commission := decimal.NewFromFloat(0.001)
		slippage := decimal.NewFromFloat(0.002)
		params := ExitParams{
			TakeProfit:     decimal.NewFromFloat(0.10),
			StopLoss:       decimal.NewFromFloat(0.05),
			MaxHoldDays:    10,
			CommissionRate: commission,
			SlippageRate:   slippage,
		}
		prices := pricesFromCloses(100, 100, 110)
		res := RunBacktest(prices, boolsAt(3, 1), nil, params)

		// 手計算
		one := decimal.NewFromInt(1)
		entryClose := decimal.NewFromFloat(100)
		exitClose := decimal.NewFromFloat(110)
		effEntry := entryClose.Mul(one.Add(slippage)).Mul(one.Add(commission))
		effExit := exitClose.Mul(one.Sub(slippage)).Mul(one.Sub(commission))
		expectedRet := effExit.Div(effEntry).Sub(one)

		assert.Equal(t, 1, res.Trades)
		retGot, _ := res.TradeList[0].Return.Float64()
		retExp, _ := expectedRet.Float64()
		assert.InDelta(t, retExp, retGot, 1e-6, "コストあり1トレードリターン（Round(6)分の誤差を許容）")

		// コストなしより低いリターンになること
		paramsNoCost := ExitParams{TakeProfit: decimal.NewFromFloat(0.10), StopLoss: decimal.NewFromFloat(0.05), MaxHoldDays: 10}
		resNoCost := RunBacktest(prices, boolsAt(3, 1), nil, paramsNoCost)
		assert.True(t, res.TradeList[0].Return.LessThan(resNoCost.TradeList[0].Return), "コストあり < コストなし")
	})

	t.Run("コストありで勝率/PFが変化する", func(t *testing.T) {
		// コストが大きいと、薄い利益は損失に転じて勝率が下がる
		// entry@100, exit@100.5（+0.5%）: コストなしなら勝ち、高コストなら負け
		commission := decimal.NewFromFloat(0.003)
		slippage := decimal.NewFromFloat(0.003)
		paramsCost := ExitParams{
			TakeProfit:     decimal.NewFromFloat(0.50),
			StopLoss:       decimal.NewFromFloat(0.50),
			MaxHoldDays:    1,
			CommissionRate: commission,
			SlippageRate:   slippage,
		}
		paramsNoCost := ExitParams{
			TakeProfit:  decimal.NewFromFloat(0.50),
			StopLoss:    decimal.NewFromFloat(0.50),
			MaxHoldDays: 1,
		}
		// 0.5% 上昇 → コストなしなら勝ち、コストあり（片道0.6%）なら負け
		prices := pricesFromCloses(100, 100, 100.5)
		signals := boolsAt(3, 1)

		resCost := RunBacktest(prices, signals, nil, paramsCost)
		resNoCost := RunBacktest(prices, signals, nil, paramsNoCost)

		assert.Equal(t, 1, resCost.Trades)
		assert.Equal(t, 1, resNoCost.Trades)
		// コストなしは勝ち（WinRate=1）、コストありは負け（WinRate=0）
		assert.True(t, resNoCost.WinRate.Equal(decimal.NewFromInt(1)), "コストなし: 勝率1")
		assert.True(t, resCost.WinRate.Equal(decimal.Zero), "コストあり: 勝率0")
	})
}

// TestRunBacktest_WithExitSignals シグナルイグジット機能のテスト
func TestRunBacktest_WithExitSignals(t *testing.T) {
	params := ExitParams{
		TakeProfit:  decimal.NewFromFloat(0.50), // 広め（シグナルが先に発火するよう）
		StopLoss:    decimal.NewFromFloat(0.50),
		MaxHoldDays: 30,
	}

	t.Run("exitSignals=nil なら従来動作と完全一致", func(t *testing.T) {
		prices := pricesFromCloses(100, 100, 102, 104, 106)
		signals := boolsAt(5, 1)

		withNil := RunBacktest(prices, signals, nil, params)
		withoutExitSignals := runBacktest(prices, signals, nil, params, true)

		assert.Equal(t, withNil.Trades, withoutExitSignals.Trades)
		assert.True(t, withNil.TotalReturn.Equal(withoutExitSignals.TotalReturn), "TotalReturn 一致")
	})

	t.Run("シグナルイグジットで TP/SL 到達前に手仕舞い", func(t *testing.T) {
		// day1でエントリー、day3で exitSignal が立つ（TP/SL未到達）
		prices := pricesFromCloses(100, 100, 101, 102, 103, 110)
		entrySignals := boolsAt(6, 1)
		exitSignals := boolsAt(6, 3) // day3で手仕舞い

		res := RunBacktest(prices, entrySignals, exitSignals, params)

		assert.Equal(t, 1, res.Trades)
		assert.Equal(t, "signal_exit", res.TradeList[0].Reason)
		assert.Equal(t, 2, res.TradeList[0].HoldDays) // day1→day3 = 2日
		// リターン = 102/100 - 1 = 0.02
		assert.InDelta(t, 0.02, f64FromDec(res.TradeList[0].Return), 1e-9)
	})

	t.Run("同日に TP とシグナルが重なったら TP 優先", func(t *testing.T) {
		// day1でエントリー、day2で TP 到達（+50%）、同日に exitSignal も true
		params2 := ExitParams{TakeProfit: decimal.NewFromFloat(0.49), StopLoss: decimal.NewFromFloat(0.50), MaxHoldDays: 30}
		prices := pricesFromCloses(100, 100, 150, 200) // day2で +50%
		entrySignals := boolsAt(4, 1)
		exitSignals := boolsAt(4, 2) // day2が TP 到達日と同日

		res := RunBacktest(prices, entrySignals, exitSignals, params2)

		assert.Equal(t, 1, res.Trades)
		// TP が優先されるべき
		assert.Equal(t, "take_profit", res.TradeList[0].Reason)
	})

	t.Run("exitSignals が entrySignals より短い場合もパニックしない", func(t *testing.T) {
		prices := pricesFromCloses(100, 100, 101, 102, 103)
		entrySignals := boolsAt(5, 1)
		exitSignals := boolsAt(3, 2) // 長さが prices より短い

		assert.NotPanics(t, func() {
			RunBacktest(prices, entrySignals, exitSignals, params)
		})
	})
}

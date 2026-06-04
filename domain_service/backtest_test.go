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
		res := RunBacktest(prices, boolsAt(3, 1), params)
		assert.Equal(t, 1, res.Trades)
		assert.InDelta(t, 0.10, f64FromDec(res.TotalReturn), 1e-9)
		assert.InDelta(t, 1.0, f64FromDec(res.WinRate), 1e-9)
		assert.Equal(t, "take_profit", res.TradeList[0].Reason)
		assert.Len(t, res.Equity, 3)
	})

	t.Run("損切りで手仕舞い", func(t *testing.T) {
		prices := pricesFromCloses(100, 100, 94)
		res := RunBacktest(prices, boolsAt(3, 1), params)
		assert.Equal(t, 1, res.Trades)
		assert.InDelta(t, -0.06, f64FromDec(res.TotalReturn), 1e-9)
		assert.InDelta(t, 0.0, f64FromDec(res.WinRate), 1e-9)
		assert.Equal(t, "stop_loss", res.TradeList[0].Reason)
		assert.InDelta(t, -0.06, f64FromDec(res.MaxDrawdown), 1e-9)
	})

	t.Run("最大保有日数で手仕舞い", func(t *testing.T) {
		p := ExitParams{TakeProfit: decimal.NewFromFloat(0.5), StopLoss: decimal.NewFromFloat(0.5), MaxHoldDays: 2}
		prices := pricesFromCloses(100, 100, 101, 102, 103)
		res := RunBacktest(prices, boolsAt(5, 1), p)
		assert.Equal(t, 1, res.Trades)
		assert.Equal(t, "max_hold", res.TradeList[0].Reason)
		assert.Equal(t, 2, res.TradeList[0].HoldDays)
		assert.InDelta(t, 0.02, f64FromDec(res.TradeList[0].Return), 1e-9) // 102/100-1
	})

	t.Run("データ末尾で強制クローズ", func(t *testing.T) {
		p := ExitParams{TakeProfit: decimal.NewFromFloat(0.5), StopLoss: decimal.NewFromFloat(0.5), MaxHoldDays: 10}
		prices := pricesFromCloses(100, 100, 101)
		res := RunBacktest(prices, boolsAt(3, 1), p)
		assert.Equal(t, 1, res.Trades)
		assert.Equal(t, "end_of_data", res.TradeList[0].Reason)
		assert.InDelta(t, 0.01, f64FromDec(res.TotalReturn), 1e-9)
	})

	t.Run("シグナルなしは取引0件", func(t *testing.T) {
		prices := pricesFromCloses(100, 101, 102)
		res := RunBacktest(prices, boolsAt(3), params)
		assert.Equal(t, 0, res.Trades)
		assert.True(t, res.TotalReturn.IsZero())
		assert.Len(t, res.Equity, 3)
	})

	t.Run("保有中は新規エントリーしない", func(t *testing.T) {
		// day1でエントリー、day2にもシグナルがあるが保有中なので無視、day3でTP
		prices := pricesFromCloses(100, 100, 105, 110)
		res := RunBacktest(prices, boolsAt(4, 1, 2), params)
		assert.Equal(t, 1, res.Trades) // 2回目のシグナルは無視される
	})

	t.Run("空入力やサイズ不一致は空結果", func(t *testing.T) {
		assert.Equal(t, 0, RunBacktest(nil, nil, params).Trades)
		assert.Equal(t, 0, RunBacktest(pricesFromCloses(100, 101), boolsAt(3), params).Trades)
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

	full := RunBacktest(prices, signals, params)
	metrics := RunBacktestMetrics(prices, signals, params)

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

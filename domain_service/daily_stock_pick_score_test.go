package domain_service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeByBreakpoints(t *testing.T) {
	points := []scoreBreakpoint{bp("0", "0"), bp("10", "1")}

	t.Run("下限未満は端点にクランプ", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(-5), points)
		assert.True(t, got.IsZero())
	})
	t.Run("上限超過は端点にクランプ", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(100), points)
		assert.True(t, got.Equal(decimal.NewFromInt(1)))
	})
	t.Run("折れ点ちょうど", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(10), points)
		assert.True(t, got.Equal(decimal.NewFromInt(1)))
	})
	t.Run("中間は線形補間", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(5), points)
		assert.True(t, got.Equal(decimal.RequireFromString("0.5")))
	})
	t.Run("同一Xが連続してもpanicしない", func(t *testing.T) {
		pts := []scoreBreakpoint{bp("0", "0"), bp("5", "0.2"), bp("5", "0.8"), bp("10", "1")}
		assert.NotPanics(t, func() {
			normalizeByBreakpoints(decimal.NewFromInt(5), pts)
		})
	})
	t.Run("pointsが1点", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(5), []scoreBreakpoint{bp("3", "0.7")})
		assert.True(t, got.Equal(decimal.RequireFromString("0.7")))
	})
	t.Run("pointsが空", func(t *testing.T) {
		got := normalizeByBreakpoints(decimal.NewFromInt(5), nil)
		assert.True(t, got.IsZero())
	})
}

func TestScoreDailyPick(t *testing.T) {
	w := DefaultDailyPickScoreWeights()

	t.Run("全因子最大 → 100.00", func(t *testing.T) {
		m := DailyPickMetrics{
			SignalCount:     4,
			VolumeRatio:     decimal.NewFromInt(10),
			ADX:             decimal.NewFromInt(50),
			ATRRatio:        decimal.RequireFromString("0.03"),
			AvgTradingValue: decimal.NewFromInt(10_000_000_000),
			RSI:             decimal.NewFromInt(60),
		}
		got := ScoreDailyPick(m, w)
		assert.Equal(t, "100.00", got.StringFixed(2))
	})

	t.Run("実運用上の最小（SignalCount>=1が保証される前提） → 9.00", func(t *testing.T) {
		m := DailyPickMetrics{
			SignalCount:     1,
			VolumeRatio:     decimal.NewFromInt(1),
			ADX:             decimal.NewFromInt(10),
			ATRRatio:        decimal.RequireFromString("0.005"),
			AvgTradingValue: decimal.NewFromInt(50_000_000),
			RSI:             decimal.NewFromInt(90),
		}
		got := ScoreDailyPick(m, w)
		assert.Equal(t, "9.00", got.StringFixed(2))
	})

	t.Run("重み差し替え（Signalのみ100%）", func(t *testing.T) {
		m := DailyPickMetrics{
			SignalCount:     4,
			VolumeRatio:     decimal.NewFromInt(10),
			ADX:             decimal.NewFromInt(50),
			ATRRatio:        decimal.RequireFromString("0.03"),
			AvgTradingValue: decimal.NewFromInt(10_000_000_000),
			RSI:             decimal.NewFromInt(60),
		}
		customW := DailyPickScoreWeights{Signal: decimal.NewFromInt(100)}
		got := ScoreDailyPick(m, customW)
		assert.Equal(t, "100.00", got.StringFixed(2))
	})

	t.Run("中間値を手計算で固定", func(t *testing.T) {
		m := DailyPickMetrics{
			SignalCount:     2,                                  // 0.65
			VolumeRatio:     decimal.NewFromInt(1),              // 0
			ADX:             decimal.NewFromInt(15),             // 0
			ATRRatio:        decimal.RequireFromString("0.010"), // 0
			AvgTradingValue: decimal.NewFromInt(100_000_000),    // 0
			RSI:             decimal.NewFromInt(30),             // 0.40
		}
		// score = 30*0.65 + 10*0.40 = 19.5 + 4 = 23.5
		got := ScoreDailyPick(m, w)
		assert.Equal(t, "23.50", got.StringFixed(2))
	})
}

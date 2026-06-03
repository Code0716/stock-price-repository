package domain_service

import (
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEntrySignalsByStrategy_LengthAndDefault(t *testing.T) {
	prices := pricesFromCloses(make([]float64, 90)...) // 全て0.0でも長さ確認には十分
	for _, s := range StrategyOrder {
		got := EntrySignalsByStrategy(s, prices)
		assert.Len(t, got, len(prices), "strategy %s", s)
	}
	// 未知の戦略は全 false
	got := EntrySignalsByStrategy("unknown", prices)
	assert.Len(t, got, len(prices))
	for _, v := range got {
		assert.False(t, v)
	}
}

func TestSmaSeries(t *testing.T) {
	closes := decs(1, 2, 3, 4, 5)
	sma := smaSeries(closes, 3)
	assert.True(t, sma[0].IsZero())
	assert.True(t, sma[1].IsZero())
	assert.InDelta(t, 2.0, f64FromDec(sma[2]), 1e-9) // (1+2+3)/3
	assert.InDelta(t, 3.0, f64FromDec(sma[3]), 1e-9)
	assert.InDelta(t, 4.0, f64FromDec(sma[4]), 1e-9)
}

func TestAvgVolume(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]*models.StockBrandDailyPrice, 6)
	vols := []int64{10, 20, 30, 40, 50, 999}
	for i, v := range vols {
		prices[i] = &models.StockBrandDailyPrice{Date: base.AddDate(0, 0, i), Volume: v}
	}
	// index5 の直前5日 (idx0..4) = (10+20+30+40+50)/5 = 30
	assert.InDelta(t, 30.0, f64FromDec(avgVolume(prices, 5, 5)), 1e-9)
}

func TestLocalExtremaSlope(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	highs := []float64{1, 5, 1, 4, 1, 3, 1} // 極大が 5,4,3 と下降
	lows := []float64{9, 1, 9, 2, 9, 3, 9}   // 極小が 1,2,3 と上昇
	window := make([]*models.StockBrandDailyPrice, len(highs))
	for i := range highs {
		window[i] = &models.StockBrandDailyPrice{
			Date: base.AddDate(0, 0, i),
			High: decimal.NewFromFloat(highs[i]),
			Low:  decimal.NewFromFloat(lows[i]),
		}
	}
	highSlope, okH := localExtremaSlope(window, true)
	lowSlope, okL := localExtremaSlope(window, false)
	assert.True(t, okH)
	assert.True(t, okL)
	assert.True(t, highSlope.IsNegative(), "高値の傾きは負")
	assert.True(t, lowSlope.IsPositive(), "安値の傾きは正")

	// 極値が2点未満なら ok=false
	flat := []*models.StockBrandDailyPrice{
		{High: decimal.NewFromInt(1), Low: decimal.NewFromInt(1)},
		{High: decimal.NewFromInt(1), Low: decimal.NewFromInt(1)},
	}
	_, ok := localExtremaSlope(flat, true)
	assert.False(t, ok)
}

func TestMovingAverageCrossEntrySignals(t *testing.T) {
	// 79日フラット(100)→最終日に200へ急騰。最終日だけ5/25/75を全上抜け。
	closes := make([]float64, 80)
	for i := 0; i < 79; i++ {
		closes[i] = 100
	}
	closes[79] = 200
	signals := MovingAverageCrossEntrySignals(pricesFromCloses(closes...))

	assert.True(t, signals[79], "最終日に全MA上抜け")
	count := 0
	for _, s := range signals {
		if s {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

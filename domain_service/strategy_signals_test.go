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

// TestExitSignalsByStrategy_LengthAndDefault ExitSignalsByStrategy の基本動作を確認。
func TestExitSignalsByStrategy_LengthAndDefault(t *testing.T) {
	prices := pricesFromCloses(make([]float64, 90)...)
	for _, s := range StrategyOrder {
		got := ExitSignalsByStrategy(s, prices)
		assert.Len(t, got, len(prices), "strategy %s", s)
	}
	// 未知の戦略は全 false
	got := ExitSignalsByStrategy("unknown", prices)
	assert.Len(t, got, len(prices))
	for _, v := range got {
		assert.False(t, v)
	}
}

// TestMACDBullishExitSignals MACD デッドクロスを検出することを確認。
func TestMACDBullishExitSignals(t *testing.T) {
	t.Run("短データ（ウォームアップ不足）は全 false", func(t *testing.T) {
		prices := pricesFromCloses(make([]float64, 30)...)
		sigs := MACDBullishExitSignals(prices)
		for _, v := range sigs {
			assert.False(t, v)
		}
	})

	t.Run("急騰後に急落するデータではデッドクロスが発生する", func(t *testing.T) {
		// 100日データ: 10日フラット + 50日急上昇 + 40日急落
		// このパターンだと minIdx=35 以降でデッドクロスが発生する
		closes := make([]float64, 100)
		for i := 0; i < 10; i++ {
			closes[i] = 100 // フラット
		}
		for i := 10; i < 60; i++ {
			closes[i] = float64(100 + (i-10)*3) // 急上昇
		}
		peak := 100 + 50*3 // 250
		for i := 60; i < 100; i++ {
			closes[i] = float64(peak - (i-60)*5) // 急落
		}
		prices := pricesFromCloses(closes...)
		sigs := MACDBullishExitSignals(prices)
		// 急落フェーズでデッドクロスが1回以上発生するはず
		count := 0
		for _, v := range sigs {
			if v {
				count++
			}
		}
		assert.Greater(t, count, 0, "急落フェーズでデッドクロスが発生するはず")
	})
}

// TestBollingerBreakoutExitSignals ミドルバンド下抜けを検出することを確認。
func TestBollingerBreakoutExitSignals(t *testing.T) {
	t.Run("短データ（ウォームアップ不足）は全 false", func(t *testing.T) {
		prices := pricesFromCloses(make([]float64, 15)...)
		sigs := BollingerBreakoutExitSignals(prices)
		for _, v := range sigs {
			assert.False(t, v)
		}
	})

	t.Run("ミドルバンドを下抜けた日が検出される", func(t *testing.T) {
		// 25日フラット(100)の後、1日だけ80に急落し翌日100に戻す
		closes := make([]float64, 25)
		for i := range closes {
			closes[i] = 100
		}
		closes = append(closes, 80) // index 25: 100→80（ミドルバンド≒100を下抜け）
		closes = append(closes, 100)
		prices := pricesFromCloses(closes...)
		sigs := BollingerBreakoutExitSignals(prices)
		assert.True(t, sigs[25], "index25 でミドルバンド下抜けシグナルが立つべき")
	})
}

// TestMovingAverageCrossExitSignals 5SMAが25SMAを下抜けを検出することを確認。
func TestMovingAverageCrossExitSignals(t *testing.T) {
	t.Run("短データは全 false", func(t *testing.T) {
		prices := pricesFromCloses(make([]float64, 10)...)
		sigs := MovingAverageCrossExitSignals(prices)
		for _, v := range sigs {
			assert.False(t, v)
		}
	})

	t.Run("5SMAが25SMAを下抜けた日が検出される", func(t *testing.T) {
		// 30日フラット(100)の後、5日連続で急落（5SMAが25SMAを下抜け）
		closes := make([]float64, 30)
		for i := range closes {
			closes[i] = 100
		}
		for i := 0; i < 5; i++ {
			closes = append(closes, 50) // 急落（5SMA≒50 < 25SMA≒100近辺）
		}
		prices := pricesFromCloses(closes...)
		sigs := MovingAverageCrossExitSignals(prices)
		// 急落フェーズで少なくとも1回下抜けが検出されるはず
		count := 0
		for _, v := range sigs {
			if v {
				count++
			}
		}
		assert.Greater(t, count, 0, "急落後に5SMA < 25SMAの下抜けが検出されるはず")
	})
}

// TestGenericTrendBreakExitSignals 終値が25SMAを下抜けを検出することを確認。
func TestGenericTrendBreakExitSignals(t *testing.T) {
	t.Run("短データは全 false", func(t *testing.T) {
		prices := pricesFromCloses(make([]float64, 10)...)
		sigs := GenericTrendBreakExitSignals(prices)
		for _, v := range sigs {
			assert.False(t, v)
		}
	})

	t.Run("終値が25SMAを下抜けた日が検出される", func(t *testing.T) {
		// 30日フラット(100)の後1日だけ50に急落（25SMA≒100を下抜け）
		closes := make([]float64, 30)
		for i := range closes {
			closes[i] = 100
		}
		closes = append(closes, 50) // index 30: 25SMAより大幅に下
		prices := pricesFromCloses(closes...)
		sigs := GenericTrendBreakExitSignals(prices)
		assert.True(t, sigs[30], "index30 で25SMA下抜けシグナルが立つべき")
	})
}

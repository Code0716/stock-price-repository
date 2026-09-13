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
	lows := []float64{9, 1, 9, 2, 9, 3, 9}  // 極小が 1,2,3 と上昇
	window := make([]*models.StockBrandDailyPrice, len(highs))
	for i := range highs {
		window[i] = &models.StockBrandDailyPrice{
			Date: base.AddDate(0, 0, i),
			High: decimal.NewFromFloat(highs[i]),
			Low:  decimal.NewFromFloat(lows[i]),
		}
	}
	highSlope, okH := localExtremaSlope(window, true, 1, 2)
	lowSlope, okL := localExtremaSlope(window, false, 1, 2)
	assert.True(t, okH)
	assert.True(t, okL)
	assert.True(t, highSlope.IsNegative(), "高値の傾きは負")
	assert.True(t, lowSlope.IsPositive(), "安値の傾きは正")

	// 極値が minPoints 未満なら ok=false
	flat := []*models.StockBrandDailyPrice{
		{High: decimal.NewFromInt(1), Low: decimal.NewFromInt(1)},
		{High: decimal.NewFromInt(1), Low: decimal.NewFromInt(1)},
	}
	_, ok := localExtremaSlope(flat, true, 1, 2)
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

// triangleTestBar 三角持ち合いテスト用の1本のOHLCVパラメータ。
type triangleTestBar struct {
	high, low, close float64
	volume           int64
}

func trianglePricesFromBars(bars []triangleTestBar) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, len(bars))
	for i, b := range bars {
		out[i] = &models.StockBrandDailyPrice{
			Date:   base.AddDate(0, 0, i),
			High:   decimal.NewFromFloat(b.high),
			Low:    decimal.NewFromFloat(b.low),
			Close:  decimal.NewFromFloat(b.close),
			Volume: b.volume,
		}
	}
	return out
}

// makeConvergingTriangleWindow 三角持ち合い（高値切り下げ・安値切り上げ・日中レンジ収縮）の
// days本を生成する。ピボット高値・安値そのものが収束し、日中レンジ(High-Low)もgapに比例して縮小する。
func makeConvergingTriangleWindow(days int, volume int64) []triangleTestBar {
	const (
		highStart, highEnd = 120.0, 101.0
		lowStart, lowEnd   = 80.0, 99.0
		period             = 4
	)
	bars := make([]triangleTestBar, days)
	for j := 0; j < days; j++ {
		progress := float64(j) / float64(days-1)
		envHigh := highStart + (highEnd-highStart)*progress
		envLow := lowStart + (lowEnd-lowStart)*progress
		gap := envHigh - envLow
		mid := (envHigh + envLow) / 2
		switch j % period {
		case 0: // 高値ピボット
			bars[j] = triangleTestBar{high: envHigh, low: envHigh - 0.1*gap, close: mid, volume: volume}
		case 2: // 安値ピボット
			bars[j] = triangleTestBar{high: envLow + 0.1*gap, low: envLow, close: mid, volume: volume}
		default: // 遷移日
			bars[j] = triangleTestBar{high: mid, low: mid - 0.1*gap, close: mid, volume: volume}
		}
	}
	return bars
}

// makeFlatRangeTriangleWindow ピボット高値・安値は収束するが、日中レンジ(High-Low)が縮小しない
// days本を生成する（「収縮しているように見えて実は縮んでいない」ケースの検証用）。
func makeFlatRangeTriangleWindow(days int, volume int64) []triangleTestBar {
	const (
		highStart, highEnd = 120.0, 101.0
		lowStart, lowEnd   = 80.0, 99.0
		period             = 4
		dayRange           = 0.5
	)
	bars := make([]triangleTestBar, days)
	for j := 0; j < days; j++ {
		progress := float64(j) / float64(days-1)
		envHigh := highStart + (highEnd-highStart)*progress
		envLow := lowStart + (lowEnd-lowStart)*progress
		mid := (envHigh + envLow) / 2
		switch j % period {
		case 0:
			bars[j] = triangleTestBar{high: envHigh, low: envHigh - dayRange, close: mid, volume: volume}
		case 2:
			bars[j] = triangleTestBar{high: envLow + dayRange, low: envLow, close: mid, volume: volume}
		default:
			bars[j] = triangleTestBar{high: mid, low: mid - dayRange, close: mid, volume: volume}
		}
	}
	return bars
}

// makeFlatSlopeTriangleWindow 高値・安値の包絡線がまったく収束しない（傾き0）days本を生成する。
func makeFlatSlopeTriangleWindow(days int, volume int64) []triangleTestBar {
	const (
		envHigh, envLow = 105.0, 95.0
		period          = 4
	)
	gap := envHigh - envLow
	mid := (envHigh + envLow) / 2
	bars := make([]triangleTestBar, days)
	for j := 0; j < days; j++ {
		switch j % period {
		case 0:
			bars[j] = triangleTestBar{high: envHigh, low: envHigh - 0.1*gap, close: mid, volume: volume}
		case 2:
			bars[j] = triangleTestBar{high: envLow + 0.1*gap, low: envLow, close: mid, volume: volume}
		default:
			bars[j] = triangleTestBar{high: mid, low: mid - 0.1*gap, close: mid, volume: volume}
		}
	}
	return bars
}

// makeMonotonicTriangleWindow 振動のない単調な包絡線（局所極値が生じない）days本を生成する。
func makeMonotonicTriangleWindow(days int, volume int64) []triangleTestBar {
	const (
		highStart, highEnd = 120.0, 101.0
		lowStart, lowEnd   = 80.0, 99.0
	)
	bars := make([]triangleTestBar, days)
	for j := 0; j < days; j++ {
		progress := float64(j) / float64(days-1)
		high := highStart + (highEnd-highStart)*progress
		low := lowStart + (lowEnd-lowStart)*progress
		bars[j] = triangleTestBar{high: high, low: low, close: (high + low) / 2, volume: volume}
	}
	return bars
}

// TestTriangleFormationEntrySignals 三角持ち合いブレイク（収縮+ブレイク+出来高）の検出を確認する。
func TestTriangleFormationEntrySignals(t *testing.T) {
	breakoutBar := triangleTestBar{high: 131, low: 129, close: 130, volume: 2000}
	noBreakoutBar := triangleTestBar{high: 100.5, low: 99.5, close: 100, volume: 2000}
	lowVolumeBreakoutBar := triangleTestBar{high: 131, low: 129, close: 130, volume: 1000}

	t.Run("収縮+当日ブレイク+出来高十分 → true", func(t *testing.T) {
		bars := makeConvergingTriangleWindow(60, 1000)
		bars = append(bars, breakoutBar)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.Len(t, signals, 61)
		assert.True(t, signals[60])
	})

	t.Run("収縮しているがブレイクしていない → false", func(t *testing.T) {
		bars := makeConvergingTriangleWindow(60, 1000)
		bars = append(bars, noBreakoutBar)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.False(t, signals[60])
	})

	t.Run("ブレイクしたが出来高不足 → false", func(t *testing.T) {
		bars := makeConvergingTriangleWindow(60, 1000)
		bars = append(bars, lowVolumeBreakoutBar)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.False(t, signals[60])
	})

	t.Run("ピボットは収束するが日中レンジが収縮していない ＋ 上抜け → false", func(t *testing.T) {
		bars := makeFlatRangeTriangleWindow(60, 1000)
		bars = append(bars, breakoutBar)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.False(t, signals[60])
	})

	t.Run("傾きが微小（ノイズレベル） → false", func(t *testing.T) {
		bars := makeFlatSlopeTriangleWindow(60, 1000)
		bars = append(bars, triangleTestBar{high: 110, low: 108, close: 109, volume: 2000})
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.False(t, signals[60])
	})

	t.Run("連日上抜けでも2日目以降は false（瞬間判定）", func(t *testing.T) {
		bars := makeConvergingTriangleWindow(60, 1000)
		bars = append(bars, breakoutBar)
		bars = append(bars, triangleTestBar{high: 136, low: 134, close: 135, volume: 2000})
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.True(t, signals[60])
		assert.False(t, signals[61])
	})

	t.Run("極値が3点未満（単調な包絡線） → false", func(t *testing.T) {
		bars := makeMonotonicTriangleWindow(60, 1000)
		bars = append(bars, breakoutBar)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.False(t, signals[60])
	})

	t.Run("バー数不足（Window未満） → 全 false", func(t *testing.T) {
		bars := makeConvergingTriangleWindow(50, 1000)
		signals := TriangleFormationEntrySignals(trianglePricesFromBars(bars))
		assert.Len(t, signals, 50)
		for _, v := range signals {
			assert.False(t, v)
		}
	})
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

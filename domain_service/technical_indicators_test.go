package domain_service

import (
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// pricesFromOHLCV テスト用に OHLCV の日足スライスを組み立てるヘルパー。
// 各引数は [high, low, close, volume] の4要素スライス。
func pricesFromOHLCV(rows ...[]float64) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, 0, len(rows))
	for i, r := range rows {
		var h, l, c float64
		var v int64
		if len(r) > 0 {
			h = r[0]
		}
		if len(r) > 1 {
			l = r[1]
		}
		if len(r) > 2 {
			c = r[2]
		}
		if len(r) > 3 {
			v = int64(r[3])
		}
		out = append(out, &models.StockBrandDailyPrice{
			Date:   base.AddDate(0, 0, i),
			High:   decimal.NewFromFloat(h),
			Low:    decimal.NewFromFloat(l),
			Close:  decimal.NewFromFloat(c),
			Volume: v,
		})
	}
	return out
}

// --- ATR ---

func TestCalculateATR(t *testing.T) {
	t.Run("データ不足(<=period)はnil", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{105, 95, 100, 1000},
			[]float64{106, 96, 101, 1000},
		)
		assert.Nil(t, CalculateATR(prices, 14))
	})

	t.Run("period=0はnil", func(t *testing.T) {
		prices := pricesFromOHLCV([]float64{100, 90, 95, 1000})
		assert.Nil(t, CalculateATR(prices, 0))
	})

	t.Run("横ばい(H=L=C)はATR=0", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{100, 100, 100, 1000}
		}
		prices := pricesFromOHLCV(rows...)
		atr := CalculateATR(prices, 14)
		assert.NotNil(t, atr)
		// index < 14 は Zero（ウォームアップ未確定）
		assert.True(t, atr[5].IsZero())
		// index 14 以降 = TR がすべて 0 なので ATR も 0
		assert.True(t, atr[14].IsZero())
	})

	t.Run("正常系: 長さが一致し index<period は Zero", func(t *testing.T) {
		rows := make([][]float64, 30)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		prices := pricesFromOHLCV(rows...)
		atr := CalculateATR(prices, 14)
		assert.NotNil(t, atr)
		assert.Equal(t, len(prices), len(atr))
		// index 0..13 は未確定(Zero)
		for i := range 14 {
			assert.True(t, atr[i].IsZero(), "index %d should be zero", i)
		}
		// index 14 以降は正の値
		assert.True(t, atr[14].GreaterThan(decimal.Zero))
	})
}

// --- Stochastics ---

func TestCalculateStochastics(t *testing.T) {
	t.Run("データ不足はnil", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{105, 95, 100, 1000},
			[]float64{106, 96, 101, 1000},
		)
		assert.Nil(t, CalculateStochastics(prices, 14, 3))
	})

	t.Run("period=0はnil", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		assert.Nil(t, CalculateStochastics(pricesFromOHLCV(rows...), 0, 3))
	})

	t.Run("横ばい(H=L=C)はK=0(ゼロ除算回避)", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{100, 100, 100, 1000}
		}
		stoch := CalculateStochastics(pricesFromOHLCV(rows...), 14, 3)
		assert.NotNil(t, stoch)
		assert.True(t, stoch[16].K.IsZero())
	})

	t.Run("正常系: K=100(Close=High)", func(t *testing.T) {
		// Close = High, Low = 0 → %K = 100
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{100, 0, 100, 1000}
		}
		stoch := CalculateStochastics(pricesFromOHLCV(rows...), 14, 3)
		assert.NotNil(t, stoch)
		// index 13 は最初に K が確定する
		assert.True(t, stoch[13].K.Equal(decimal.NewFromInt(100)), "K should be 100")
	})

	t.Run("正常系: 長さが一致", func(t *testing.T) {
		rows := make([][]float64, 30)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		stoch := CalculateStochastics(pricesFromOHLCV(rows...), 14, 3)
		assert.NotNil(t, stoch)
		assert.Equal(t, 30, len(stoch))
	})
}

// --- ADX ---

func TestCalculateADX(t *testing.T) {
	t.Run("データ不足(< 2*period)はnil", func(t *testing.T) {
		rows := make([][]float64, 20) // 14*2=28 必要
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		assert.Nil(t, CalculateADX(pricesFromOHLCV(rows...), 14))
	})

	t.Run("period=0はnil", func(t *testing.T) {
		rows := make([][]float64, 30)
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		assert.Nil(t, CalculateADX(pricesFromOHLCV(rows...), 0))
	})

	t.Run("正常系: 長さ一致・ADX は index>=27 で非Zero", func(t *testing.T) {
		rows := make([][]float64, 50)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		adx := CalculateADX(pricesFromOHLCV(rows...), 14)
		assert.NotNil(t, adx)
		assert.Equal(t, 50, len(adx))
		// ADX は index 27（=2*14-1）から確定
		assert.True(t, adx[26].ADX.IsZero(), "index 26 ADX should be zero")
		assert.True(t, adx[27].ADX.GreaterThan(decimal.Zero), "index 27 ADX should be positive")
	})

	t.Run("横ばい: ADX=0(方向性なし)", func(t *testing.T) {
		rows := make([][]float64, 40)
		for i := range rows {
			rows[i] = []float64{100, 100, 100, 1000}
		}
		adx := CalculateADX(pricesFromOHLCV(rows...), 14)
		assert.NotNil(t, adx)
		assert.True(t, adx[27].ADX.IsZero())
	})
}

// --- OBV ---

func TestCalculateOBV(t *testing.T) {
	t.Run("空スライスはnil", func(t *testing.T) {
		assert.Nil(t, CalculateOBV(nil))
		assert.Nil(t, CalculateOBV([]*models.StockBrandDailyPrice{}))
	})

	t.Run("上昇→出来高加算", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{100, 90, 100, 1000},
			[]float64{105, 95, 105, 2000}, // Close 上昇 → +2000
		)
		obv := CalculateOBV(prices)
		assert.NotNil(t, obv)
		assert.True(t, obv[1].Equal(decimal.NewFromInt(2000)))
	})

	t.Run("下落→出来高減算", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{100, 90, 100, 1000},
			[]float64{95, 85, 95, 1500}, // Close 下落 → -1500
		)
		obv := CalculateOBV(prices)
		assert.True(t, obv[1].Equal(decimal.NewFromInt(-1500)))
	})

	t.Run("変わらず→OBV変化なし", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{100, 90, 100, 1000},
			[]float64{105, 95, 100, 2000}, // Close 同じ → 据え置き
		)
		obv := CalculateOBV(prices)
		assert.True(t, obv[1].IsZero())
	})

	t.Run("累積が正しい", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{100, 90, 100, 1000},
			[]float64{105, 95, 105, 2000}, // Close 上昇 → +2000 = 2000
			[]float64{103, 93, 103, 500},  // Close 下落 → -500  = 1500
			[]float64{106, 96, 106, 300},  // Close 上昇 → +300  = 1800
		)
		obv := CalculateOBV(prices)
		assert.True(t, obv[2].Equal(decimal.NewFromInt(1500)))
		assert.True(t, obv[3].Equal(decimal.NewFromInt(1800)))
	})
}

// --- Ichimoku ---

func TestCalculateIchimoku(t *testing.T) {
	t.Run("データ不足(len < spanB)はnil", func(t *testing.T) {
		rows := make([][]float64, 30) // spanB=52 必要
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		assert.Nil(t, CalculateIchimoku(pricesFromOHLCV(rows...), 9, 26, 52))
	})

	t.Run("period=0はnil", func(t *testing.T) {
		rows := make([][]float64, 60)
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		prices := pricesFromOHLCV(rows...)
		assert.Nil(t, CalculateIchimoku(prices, 0, 26, 52))
		assert.Nil(t, CalculateIchimoku(prices, 9, 0, 52))
		assert.Nil(t, CalculateIchimoku(prices, 9, 26, 0))
	})

	t.Run("配列長がlen(prices)と一致する", func(t *testing.T) {
		rows := make([][]float64, 60)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		prices := pricesFromOHLCV(rows...)
		ich := CalculateIchimoku(prices, 9, 26, 52)
		assert.NotNil(t, ich)
		assert.Equal(t, len(prices), len(ich))
	})

	t.Run("横ばい価格: Tenkan=Kijun=SenkouA=SenkouB=価格", func(t *testing.T) {
		rows := make([][]float64, 60)
		for i := range rows {
			rows[i] = []float64{100, 100, 100, 1000}
		}
		prices := pricesFromOHLCV(rows...)
		ich := CalculateIchimoku(prices, 9, 26, 52)
		assert.NotNil(t, ich)
		expected := decimal.NewFromInt(100)
		// index 51（spanB-1）以降は全ライン確定
		assert.True(t, ich[51].Tenkan.Equal(expected), "Tenkan got %s", ich[51].Tenkan)
		assert.True(t, ich[51].Kijun.Equal(expected), "Kijun got %s", ich[51].Kijun)
		assert.True(t, ich[51].SenkouA.Equal(expected), "SenkouA got %s", ich[51].SenkouA)
		assert.True(t, ich[51].SenkouB.Equal(expected), "SenkouB got %s", ich[51].SenkouB)
	})

	t.Run("index < conv-1 は Tenkan がゼロ", func(t *testing.T) {
		rows := make([][]float64, 60)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		ich := CalculateIchimoku(pricesFromOHLCV(rows...), 9, 26, 52)
		assert.True(t, ich[7].Tenkan.IsZero(), "index 7 Tenkan should be zero")
		assert.False(t, ich[8].Tenkan.IsZero(), "index 8 Tenkan should be non-zero")
	})
}

// --- SupportResistance ---

func TestCalculateSupportResistance(t *testing.T) {
	t.Run("データ不足はnil", func(t *testing.T) {
		rows := make([][]float64, 5) // lookback=3 → 2*3+1=7 必要
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		result := CalculateSupportResistance(pricesFromOHLCV(rows...), 3, decimal.NewFromFloat(0.015))
		assert.Nil(t, result)
	})

	t.Run("lookback=0はnil", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 1000}
		}
		result := CalculateSupportResistance(pricesFromOHLCV(rows...), 0, decimal.NewFromFloat(0.015))
		assert.Nil(t, result)
	})

	t.Run("単調増加: スイング安値なし→レベルが少ない", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		result := CalculateSupportResistance(pricesFromOHLCV(rows...), 3, decimal.NewFromFloat(0.015))
		assert.NotNil(t, result)
		// 単調増加なのでスイング高値は最後の1点のみ（端を除いた範囲）
		// スイング安値は検出されない
		assert.LessOrEqual(t, len(result), 2)
	})

	t.Run("明確な天底: レベルと Touches が期待通り", func(t *testing.T) {
		// 高値100→ピーク110→谷90→ピーク112→谷88→高値105
		rows := [][]float64{
			{100, 98, 99, 1000},
			{102, 99, 101, 1000},
			{105, 100, 103, 1000},
			{110, 105, 108, 1000}, // ピーク高値
			{107, 102, 104, 1000},
			{103, 98, 100, 1000},
			{100, 90, 92, 1000},  // 谷安値
			{101, 92, 98, 1000},
			{104, 97, 102, 1000},
			{108, 103, 106, 1000},
			{112, 108, 110, 1000}, // ピーク高値
			{109, 104, 106, 1000},
			{105, 100, 102, 1000},
			{101, 88, 90, 1000},   // 谷安値
			{102, 90, 98, 1000},
			{104, 95, 101, 1000},
			{106, 99, 103, 1000},
		}
		prices := pricesFromOHLCV(rows...)
		tol := decimal.NewFromFloat(0.015)
		result := CalculateSupportResistance(prices, 3, tol)
		assert.NotNil(t, result)
		assert.Greater(t, len(result), 0)
		// レベルは price 昇順
		for i := 1; i < len(result); i++ {
			assert.True(t, result[i].Price.GreaterThanOrEqual(result[i-1].Price),
				"levels should be ascending: %s < %s", result[i-1].Price, result[i].Price)
		}
	})

	t.Run("tolerance による統合: 近接した2スイングが1レベルに集約", func(t *testing.T) {
		// 高値100(index=3)と高値101(index=7)の1%差 → tol=1.5% で統合される
		// lookback=3 なのでループは i=3..n-4
		rows := [][]float64{
			{95, 85, 90, 1000},
			{96, 86, 91, 1000},
			{97, 87, 92, 1000},
			{100, 90, 95, 1000}, // index=3: スイング高値100（[0..6]で最大）
			{97, 87, 92, 1000},
			{95, 85, 90, 1000},
			{96, 86, 91, 1000},
			{101, 91, 96, 1000}, // index=7: スイング高値101（[4..10]で最大）
			{98, 88, 93, 1000},
			{96, 86, 91, 1000},
			{95, 85, 90, 1000},
		}
		prices := pricesFromOHLCV(rows...)
		tol := decimal.NewFromFloat(0.015)
		result := CalculateSupportResistance(prices, 3, tol)
		// 100と101は1%差なので tol=1.5% で統合→Touches>=2
		hasMerged := false
		for _, lv := range result {
			if lv.Touches >= 2 {
				hasMerged = true
			}
		}
		assert.True(t, hasMerged, "近接スイングが統合されたレベルが存在するはず: %+v", result)
	})
}

// --- VWAP ---

func TestCalculateRollingVWAP(t *testing.T) {
	t.Run("データ不足(<period)はnil", func(t *testing.T) {
		prices := pricesFromOHLCV(
			[]float64{105, 95, 100, 1000},
		)
		assert.Nil(t, CalculateRollingVWAP(prices, 14))
	})

	t.Run("period=0はnil", func(t *testing.T) {
		prices := pricesFromOHLCV([]float64{100, 90, 95, 1000})
		assert.Nil(t, CalculateRollingVWAP(prices, 0))
	})

	t.Run("出来高ゼロはVWAP=0(ゼロ除算回避)", func(t *testing.T) {
		rows := make([][]float64, 5)
		for i := range rows {
			rows[i] = []float64{100, 90, 95, 0} // volume=0
		}
		vwap := CalculateRollingVWAP(pricesFromOHLCV(rows...), 5)
		assert.NotNil(t, vwap)
		assert.True(t, vwap[4].IsZero())
	})

	t.Run("正常系: 均一価格・均一出来高 → typical price に等しい", func(t *testing.T) {
		// H=105, L=95, C=100 → typical = (105+95+100)/3 = 100
		rows := make([][]float64, 5)
		for i := range rows {
			rows[i] = []float64{105, 95, 100, 1000}
		}
		vwap := CalculateRollingVWAP(pricesFromOHLCV(rows...), 5)
		assert.NotNil(t, vwap)
		expected := decimal.NewFromFloat(100) // typical price
		assert.True(t, vwap[4].Equal(expected), "got %s, want %s", vwap[4], expected)
	})

	t.Run("正常系: 長さが一致・index<period-1 はZero", func(t *testing.T) {
		rows := make([][]float64, 20)
		for i := range rows {
			rows[i] = []float64{float64(100 + i), float64(90 + i), float64(95 + i), 1000}
		}
		vwap := CalculateRollingVWAP(pricesFromOHLCV(rows...), 5)
		assert.NotNil(t, vwap)
		assert.Equal(t, 20, len(vwap))
		// index 0..3 は未確定（Zero）
		for i := range 4 {
			assert.True(t, vwap[i].IsZero(), "index %d should be zero", i)
		}
		assert.True(t, vwap[4].GreaterThan(decimal.Zero))
	})
}

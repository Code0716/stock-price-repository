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

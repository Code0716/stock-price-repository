package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
)

// avoidTestBrand 流動性を満たすテスト用ブランドを返す。
func avoidTestBrand(id, ticker string) *models.StockBrand {
	return &models.StockBrand{ID: id, TickerSymbol: ticker, Name: "テスト銘柄" + ticker, Sector33CodeName: "電気機器"}
}

// makeAvoidSeries n本の日足系列を生成する。dailyMovePct（例: 0.001=0.1%）だけ毎日交互に上下させ、
// close=1000円・volume=200,000株を基準にする（流動性ユニバース基準1億円/日を十分満たす）。
func makeAvoidSeries(n int, dailyMovePct decimal.Decimal) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, n)
	price := decimal.NewFromInt(1000)
	for i := 0; i < n; i++ {
		if i > 0 {
			move := dailyMovePct
			if i%2 == 0 {
				move = move.Neg()
			}
			price = price.Mul(decimal.NewFromInt(1).Add(move))
		}
		out[i] = dailyPickBar(base.AddDate(0, 0, i), price, price.Add(decimal.NewFromInt(1)), price.Sub(decimal.NewFromInt(1)), 200000)
	}
	return out
}

func TestEvaluateAvoidCandidate(t *testing.T) {
	filter := DefaultDailyPickFilterParams()
	brand := avoidTestBrand("brand-1", "1234")

	t.Run("十分な履歴があれば評価できる", func(t *testing.T) {
		prices := makeAvoidSeries(260, decimal.RequireFromString("0.01"))
		got := EvaluateAvoidCandidate(brand, prices, filter)
		require.NotNil(t, got)
		assert.True(t, got.Volatility12M.IsPositive())
		assert.Equal(t, brand, got.Brand)
	})

	t.Run("履歴がDailyAvoidMinReturns未満なら nil", func(t *testing.T) {
		prices := makeAvoidSeries(150, decimal.RequireFromString("0.01"))
		got := EvaluateAvoidCandidate(brand, prices, filter)
		assert.Nil(t, got)
	})

	t.Run("流動性ユニバースに入らない（株価下限未満）なら nil", func(t *testing.T) {
		prices := makeAvoidSeries(260, decimal.RequireFromString("0.01"))
		for _, p := range prices {
			p.Close = decimal.NewFromInt(100) // 300円未満
		}
		got := EvaluateAvoidCandidate(brand, prices, filter)
		assert.Nil(t, got)
	})

	t.Run("値幅ゼロ（ストップ高相当）でも評価できる（買い候補と異なりユニバース判定のみ）", func(t *testing.T) {
		prices := makeAvoidSeries(260, decimal.RequireFromString("0.01"))
		last := prices[len(prices)-1]
		last.High = last.Close
		last.Low = last.Close
		got := EvaluateAvoidCandidate(brand, prices, filter)
		assert.NotNil(t, got, "避け銘柄側は値幅ゼロを除外してはいけない（最も危険な銘柄が抜け落ちるため）")
	})

	t.Run("空スライスなら nil", func(t *testing.T) {
		got := EvaluateAvoidCandidate(brand, nil, filter)
		assert.Nil(t, got)
	})

	t.Run("ボラが高いほどVolatility12Mが大きい", func(t *testing.T) {
		low := EvaluateAvoidCandidate(brand, makeAvoidSeries(260, decimal.RequireFromString("0.001")), filter)
		high := EvaluateAvoidCandidate(brand, makeAvoidSeries(260, decimal.RequireFromString("0.05")), filter)
		require.NotNil(t, low)
		require.NotNil(t, high)
		assert.True(t, high.Volatility12M.GreaterThan(low.Volatility12M))
	})
}

func avoidCandidate(id, ticker string, vol decimal.Decimal) *AvoidCandidate {
	return &AvoidCandidate{
		Brand:           avoidTestBrand(id, ticker),
		Volatility12M:   vol,
		AvgTradingValue: decimal.NewFromInt(200000000),
		BaseClosePrice:  decimal.NewFromInt(1000),
	}
}

func TestRankAvoidCandidates(t *testing.T) {
	asOfDate := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	params := DefaultDailyAvoidFlagParams()

	t.Run("空なら nil", func(t *testing.T) {
		got := RankAvoidCandidates(nil, asOfDate, params)
		assert.Nil(t, got)
	})

	t.Run("10件中上位20%(2件)がフラグされ、上位10%(1件)がhigh", func(t *testing.T) {
		candidates := make([]*AvoidCandidate, 0, 10)
		for i := 0; i < 10; i++ {
			// vol降順で並ぶよう、iが小さいほど高ボラにする
			vol := decimal.NewFromInt(int64(10 - i))
			candidates = append(candidates, avoidCandidate("brand", "T"+decimal.NewFromInt(int64(i)).String(), vol))
		}
		got := RankAvoidCandidates(candidates, asOfDate, params)
		require.Len(t, got, 2)

		assert.Equal(t, 1, got[0].AvoidRank)
		assert.Equal(t, models.DailyAvoidSeverityHigh, got[0].Severity)
		assert.True(t, got[0].Volatility12M.Equal(decimal.NewFromInt(10)))
		assert.Equal(t, 10, got[0].UniverseSize)
		assert.True(t, got[0].VolatilityPercentile.Equal(decimal.RequireFromString("0.1")))

		assert.Equal(t, 2, got[1].AvoidRank)
		assert.Equal(t, models.DailyAvoidSeverityElevated, got[1].Severity)
		assert.True(t, got[1].Volatility12M.Equal(decimal.NewFromInt(9)))

		// 全行のUniverseSize/ThresholdVolatilityは同値
		for _, r := range got {
			assert.Equal(t, 10, r.UniverseSize)
			assert.True(t, r.ThresholdVolatility.Equal(decimal.NewFromInt(9)), "閾値はフラグ対象最下位(2位)のボラ")
			assert.Equal(t, models.DailyAvoidReasonHighVolatility, r.Reason)
			assert.Equal(t, DailyAvoidRuleVersion, r.RuleVersion)
		}
	})

	t.Run("同ボラはTickerSymbol昇順で安定ソート", func(t *testing.T) {
		vol := decimal.NewFromInt(1)
		candidates := []*AvoidCandidate{
			avoidCandidate("b2", "2000", vol),
			avoidCandidate("b1", "1000", vol),
		}
		params := DailyAvoidFlagParams{FlagPercentile: decimal.RequireFromString("1.0"), HighPercentile: decimal.RequireFromString("0.5")}
		got := RankAvoidCandidates(candidates, asOfDate, params)
		require.Len(t, got, 2)
		assert.Equal(t, "1000", got[0].TickerSymbol)
		assert.Equal(t, "2000", got[1].TickerSymbol)
	})

	t.Run("1件のみでFlagPercentile20%でも切り上げで1件フラグされる", func(t *testing.T) {
		candidates := []*AvoidCandidate{avoidCandidate("b1", "1000", decimal.NewFromInt(1))}
		got := RankAvoidCandidates(candidates, asOfDate, params)
		require.Len(t, got, 1)
	})
}

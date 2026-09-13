package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func dailyPickBar(date time.Time, close, high, low decimal.Decimal, volume int64) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		Date:     date,
		Close:    close,
		High:     high,
		Low:      low,
		Volume:   volume,
		Adjclose: close,
	}
}

// makeDailyPickSeries n本のうち最終日だけ大きく動く日足系列を生成する（最終日以外は volume で一定）。
func makeDailyPickSeries(n int, flatClose, lastClose decimal.Decimal, volume, lastVolume int64) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, n)
	for i := 0; i < n-1; i++ {
		out[i] = dailyPickBar(
			base.AddDate(0, 0, i),
			flatClose,
			flatClose.Add(decimal.NewFromInt(1)),
			flatClose.Sub(decimal.NewFromInt(1)),
			volume,
		)
	}
	out[n-1] = dailyPickBar(
		base.AddDate(0, 0, n-1),
		lastClose,
		lastClose.Add(decimal.NewFromInt(5)),
		lastClose.Sub(decimal.NewFromInt(5)),
		lastVolume,
	)
	return out
}

// makeDailyPickSeriesZeroBaselineVolume 最終日以外の出来高が全てゼロの日足系列を生成する
// （avgVolume=0 でゼロ除算panicしないことの検証用）。
func makeDailyPickSeriesZeroBaselineVolume(n int, flatClose, lastClose decimal.Decimal, lastVolume int64) []*models.StockBrandDailyPrice {
	return makeDailyPickSeries(n, flatClose, lastClose, 0, lastVolume)
}

func dailyPickTestBrand() *models.StockBrand {
	return &models.StockBrand{ID: "brand-1", TickerSymbol: "1234", Name: "テスト銘柄", Sector33CodeName: "電気機器"}
}

func TestEvaluateDailyPickCandidate(t *testing.T) {
	filter := DefaultDailyPickFilterParams()
	weights := DefaultDailyPickScoreWeights()
	brand := dailyPickTestBrand()

	t.Run("正常系: 収縮後ブレイクでMA上抜けが点灯しスコアが算出される", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(1000), decimal.NewFromInt(2000), 2_000_000, 6_000_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.NotNil(t, c)
		assert.Contains(t, c.Strategies, StrategyMovingAverageCross)
		assert.True(t, c.Score.GreaterThan(decimal.Zero))
		assert.True(t, c.Score.LessThanOrEqual(decimal.NewFromInt(100)))
		assert.Equal(t, brand, c.Brand)
	})

	t.Run("戦略が1つも点灯しない（完全に横ばい） → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(1000), decimal.NewFromInt(1000), 2_000_000, 2_000_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})

	t.Run("株価299円 → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(100), decimal.NewFromInt(299), 2_000_000, 6_000_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})

	t.Run("直近平均売買代金9000万円（1億未満） → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(1000), decimal.NewFromInt(1000), 90_000, 90_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})

	t.Run("avgVolume=0でもpanicせずVolumeRatioは0になる", func(t *testing.T) {
		prices := makeDailyPickSeriesZeroBaselineVolume(120, decimal.NewFromInt(1000), decimal.NewFromInt(2000), 3_000_000)
		var c *DailyPickCandidate
		assert.NotPanics(t, func() {
			c = EvaluateDailyPickCandidate(brand, prices, filter, weights)
		})
		if assert.NotNil(t, c) {
			assert.True(t, c.Metrics.VolumeRatio.IsZero())
		}
	})

	t.Run("-DI>+DI（下降トレンド中の急落） → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(2000), decimal.NewFromInt(1000), 2_000_000, 6_000_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})

	t.Run("当日high==low（値幅ゼロ） → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(120, decimal.NewFromInt(1000), decimal.NewFromInt(2000), 2_000_000, 6_000_000)
		last := prices[len(prices)-1]
		last.High = last.Close
		last.Low = last.Close
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})

	t.Run("バー数不足（WindowDays未満） → nil", func(t *testing.T) {
		prices := makeDailyPickSeries(100, decimal.NewFromInt(1000), decimal.NewFromInt(2000), 2_000_000, 6_000_000)
		c := EvaluateDailyPickCandidate(brand, prices, filter, weights)
		assert.Nil(t, c)
	})
}

func rankTestCandidate(id, ticker, sector, score string) *DailyPickCandidate {
	return &DailyPickCandidate{
		Brand: &models.StockBrand{ID: id, TickerSymbol: ticker, Sector33CodeName: sector},
		Score: decimal.RequireFromString(score),
	}
}

func TestRankDailyPickCandidates(t *testing.T) {
	pickDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("スコア降順・同点はTickerSymbol昇順・PickRankは1..N連番", func(t *testing.T) {
		cands := []*DailyPickCandidate{
			rankTestCandidate("b3", "3000", "sec-a", "50"),
			rankTestCandidate("b1", "1000", "sec-a", "80"),
			rankTestCandidate("b2", "2000", "sec-a", "80"),
		}
		picks := RankDailyPickCandidates(cands, pickDate, 10, 0)
		assert.Len(t, picks, 3)
		assert.Equal(t, "1000", picks[0].TickerSymbol)
		assert.Equal(t, "2000", picks[1].TickerSymbol)
		assert.Equal(t, "3000", picks[2].TickerSymbol)
		assert.Equal(t, 1, picks[0].PickRank)
		assert.Equal(t, 2, picks[1].PickRank)
		assert.Equal(t, 3, picks[2].PickRank)
	})

	t.Run("セクター上限に達すると次点が繰り上がる", func(t *testing.T) {
		cands := []*DailyPickCandidate{
			rankTestCandidate("b1", "1001", "sec-a", "90"),
			rankTestCandidate("b2", "1002", "sec-a", "85"),
			rankTestCandidate("b3", "1003", "sec-a", "80"),
			rankTestCandidate("b4", "1004", "sec-b", "75"),
		}
		picks := RankDailyPickCandidates(cands, pickDate, 3, 2)
		assert.Len(t, picks, 3)
		tickers := []string{picks[0].TickerSymbol, picks[1].TickerSymbol, picks[2].TickerSymbol}
		assert.Equal(t, []string{"1001", "1002", "1004"}, tickers)
	})

	t.Run("maxPerSector=0で無制限", func(t *testing.T) {
		cands := []*DailyPickCandidate{
			rankTestCandidate("b1", "1001", "sec-a", "90"),
			rankTestCandidate("b2", "1002", "sec-a", "85"),
			rankTestCandidate("b3", "1003", "sec-a", "80"),
		}
		picks := RankDailyPickCandidates(cands, pickDate, 3, 0)
		assert.Len(t, picks, 3)
	})

	t.Run("topNより候補が少ない場合はそのまま返す", func(t *testing.T) {
		cands := []*DailyPickCandidate{rankTestCandidate("b1", "1001", "sec-a", "90")}
		picks := RankDailyPickCandidates(cands, pickDate, 25, 4)
		assert.Len(t, picks, 1)
		assert.Equal(t, 1, picks[0].PickRank)
	})

	t.Run("セクター未設定は上限の対象外", func(t *testing.T) {
		cands := []*DailyPickCandidate{
			rankTestCandidate("b1", "1001", "", "90"),
			rankTestCandidate("b2", "1002", "", "85"),
			rankTestCandidate("b3", "1003", "", "80"),
		}
		picks := RankDailyPickCandidates(cands, pickDate, 3, 1)
		assert.Len(t, picks, 3)
	})
}

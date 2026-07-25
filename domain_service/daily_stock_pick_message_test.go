package domain_service

import (
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func messageTestPick(rank int, ticker, name, sector, score string) *models.DailyStockPick {
	return &models.DailyStockPick{
		PickDate:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StockBrandID:     "brand-" + ticker,
		TickerSymbol:     ticker,
		Name:             name,
		PickRank:         rank,
		Score:            decimal.RequireFromString(score),
		ScoreVersion:     DailyPickScoreVersion,
		SignalCount:      2,
		Strategies:       []string{StrategyMACDBullish, StrategyMovingAverageCross},
		Sector33CodeName: sector,
		BaseClosePrice:   decimal.NewFromInt(2845),
		AvgTradingValue:  decimal.NewFromInt(18_200_000_000),
		VolumeRatio:      decimal.RequireFromString("2.4"),
		ADX:              decimal.RequireFromString("31.2"),
		ATRRatio:         decimal.RequireFromString("0.028"),
		RSI:              decimal.RequireFromString("58.1"),
	}
}

func TestFormatDailyStockPickMessages(t *testing.T) {
	t.Run("空なら空文字とnil", func(t *testing.T) {
		title, bodies := FormatDailyStockPickMessages(nil, DailyPickSlackMaxRunes)
		assert.Equal(t, "", title)
		assert.Nil(t, bodies)
	})

	t.Run("25銘柄で各bodyがmaxRunes以下かつブロックが途中で切れない", func(t *testing.T) {
		picks := make([]*models.DailyStockPick, 0, 25)
		for i := 1; i <= 25; i++ {
			picks = append(picks, messageTestPick(i, "1000", "テスト銘柄株式会社ロング名称", "電気機器", "78.4"))
		}
		title, bodies := FormatDailyStockPickMessages(picks, DailyPickSlackMaxRunes)
		assert.NotEmpty(t, title)
		assert.NotEmpty(t, bodies)
		for _, b := range bodies {
			assert.LessOrEqual(t, utf8.RuneCountInString(b), DailyPickSlackMaxRunes)
		}
		// 全ての "*N.*" 見出しがどこかのbodyに過不足なく含まれていること（ブロックが分断されていない証拠）
		joined := strings.Join(bodies, "\n")
		for i := 1; i <= 25; i++ {
			assert.Contains(t, joined, "*"+strconv.Itoa(i)+".*")
		}
	})

	t.Run("&<>を含む銘柄名がエスケープされる", func(t *testing.T) {
		picks := []*models.DailyStockPick{messageTestPick(1, "1000", "A&B<C>D", "電気機器", "50")}
		_, bodies := FormatDailyStockPickMessages(picks, DailyPickSlackMaxRunes)
		joined := strings.Join(bodies, "\n")
		assert.Contains(t, joined, "A&amp;B&lt;C&gt;D")
		assert.NotContains(t, joined, "A&B<C>D")
	})

	t.Run("小さいmaxRunesでも1件は必ず出力される", func(t *testing.T) {
		picks := []*models.DailyStockPick{messageTestPick(1, "1000", "テスト銘柄", "電気機器", "50")}
		_, bodies := FormatDailyStockPickMessages(picks, 1)
		assert.NotEmpty(t, bodies)
	})
}

func TestFormatDailyStockPickResultMessage(t *testing.T) {
	t.Run("nilやTotal=0は空文字", func(t *testing.T) {
		assert.Equal(t, "", FormatDailyStockPickResultMessage(nil))
		assert.Equal(t, "", FormatDailyStockPickResultMessage(&models.DailyStockPickSummary{Total: 0}))
	})

	t.Run("勝敗と平均リターンを整形する", func(t *testing.T) {
		r1 := decimal.RequireFromString("0.087")
		s := &models.DailyStockPickSummary{
			PickDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Total:       25,
			Win:         15,
			Lose:        8,
			Draw:        1,
			Void:        1,
			WinRate:     decimal.RequireFromString("0.6522"),
			AvgReturn1D: decimal.RequireFromString("0.0042"),
			AvgReturn5D: decimal.RequireFromString("0.0183"),
			Best:        &models.DailyStockPick{TickerSymbol: "6920", Name: "レーザーテック", Return5D: &r1},
		}
		msg := FormatDailyStockPickResultMessage(s)
		assert.Contains(t, msg, "25銘柄")
		assert.Contains(t, msg, "15勝 8敗 1分 1除外")
		assert.Contains(t, msg, "65.2%")
		assert.Contains(t, msg, "+0.4%")
		assert.Contains(t, msg, "+1.8%")
		assert.Contains(t, msg, "6920")
		assert.Contains(t, msg, "+8.7%")
	})
}

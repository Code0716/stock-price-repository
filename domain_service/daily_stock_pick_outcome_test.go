package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func TestJudgeDailyPickOutcome(t *testing.T) {
	assert.Equal(t, models.DailyStockPickOutcomeWin, JudgeDailyPickOutcome(decimal.RequireFromString("0.01")))
	assert.Equal(t, models.DailyStockPickOutcomeLose, JudgeDailyPickOutcome(decimal.RequireFromString("-0.01")))
	assert.Equal(t, models.DailyStockPickOutcomeDraw, JudgeDailyPickOutcome(decimal.Zero))
}

func outcomeTestPick(outcome *models.DailyStockPickOutcome, return1D, return5D *decimal.Decimal) *models.DailyStockPick {
	return &models.DailyStockPick{
		PickDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		TickerSymbol: "1000",
		Name:         "テスト銘柄",
		Outcome:      outcome,
		Return1D:     return1D,
		Return5D:     return5D,
	}
}

func dailyPickDecPtr(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

func outcomePtr(o models.DailyStockPickOutcome) *models.DailyStockPickOutcome {
	return &o
}

func TestSummarizeDailyStockPicks(t *testing.T) {
	t.Run("空なら nil", func(t *testing.T) {
		assert.Nil(t, SummarizeDailyStockPicks(nil))
	})

	t.Run("勝敗集計・voidは勝率の分母から除外・平均リターン・ベストワースト", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeWin), dailyPickDecPtr("0.01"), dailyPickDecPtr("0.05")),
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeWin), dailyPickDecPtr("0.02"), dailyPickDecPtr("0.03")),
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeLose), dailyPickDecPtr("-0.01"), dailyPickDecPtr("-0.02")),
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeDraw), dailyPickDecPtr("0"), dailyPickDecPtr("0")),
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeVoid), nil, nil),
		}
		picks[0].TickerSymbol = "BEST"
		picks[2].TickerSymbol = "WORST"

		s := SummarizeDailyStockPicks(picks)
		assert.NotNil(t, s)
		assert.Equal(t, 5, s.Total)
		assert.Equal(t, 2, s.Win)
		assert.Equal(t, 1, s.Lose)
		assert.Equal(t, 1, s.Draw)
		assert.Equal(t, 1, s.Void)
		// 勝率の分母は win+lose+draw=4件（voidは除外）: 2/4 = 0.5
		assert.True(t, s.WinRate.Equal(decimal.RequireFromString("0.5")), "got %s", s.WinRate.String())
		assert.NotNil(t, s.Best)
		assert.Equal(t, "BEST", s.Best.TickerSymbol)
		assert.NotNil(t, s.Worst)
		assert.Equal(t, "WORST", s.Worst.TickerSymbol)
	})

	t.Run("全件voidでも0除算しない", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeVoid), nil, nil),
			outcomeTestPick(outcomePtr(models.DailyStockPickOutcomeVoid), nil, nil),
		}
		var s *models.DailyStockPickSummary
		assert.NotPanics(t, func() {
			s = SummarizeDailyStockPicks(picks)
		})
		assert.NotNil(t, s)
		assert.Equal(t, 2, s.Void)
		assert.True(t, s.WinRate.IsZero())
		assert.Nil(t, s.Best)
		assert.Nil(t, s.Worst)
	})

	t.Run("未評価（Outcome=nil）は勝敗集計されない", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			outcomeTestPick(nil, nil, nil),
		}
		s := SummarizeDailyStockPicks(picks)
		assert.NotNil(t, s)
		assert.Equal(t, 0, s.Win+s.Lose+s.Draw+s.Void)
	})
}

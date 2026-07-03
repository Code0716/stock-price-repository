package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func quizPrice(stockBrandID, ticker string, date time.Time, close, high, low decimal.Decimal, volume int64) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		StockBrandID: stockBrandID,
		TickerSymbol: ticker,
		Date:         date,
		Close:        close,
		High:         high,
		Low:          low,
		Volume:       volume,
	}
}

func TestSelectQuizUniverse(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	d0, d1, d2 := base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)
	close100 := decimal.NewFromInt(100)

	var prices []*models.StockBrandDailyPrice
	// Brand A: 出来高最大級だが値幅が小さい
	for _, d := range []time.Time{d0, d1, d2} {
		prices = append(prices, quizPrice("brand-a", "A001", d, close100, decimal.NewFromInt(101), decimal.NewFromInt(99), 1000))
	}
	// Brand B: 出来高2位・値幅最大
	for _, d := range []time.Time{d0, d1, d2} {
		prices = append(prices, quizPrice("brand-b", "B001", d, close100, decimal.NewFromInt(110), decimal.NewFromInt(90), 900))
	}
	// Brand C: 出来高3位・値幅2位
	for _, d := range []time.Time{d0, d1, d2} {
		prices = append(prices, quizPrice("brand-c", "C001", d, close100, decimal.NewFromInt(105), decimal.NewFromInt(95), 800))
	}
	// Brand D: 出来高は最大だが、期間中に欠損日がある（新規上場/売買停止想定）→ 除外されるべき
	prices = append(prices, quizPrice("brand-d", "D001", d0, close100, decimal.NewFromInt(103), decimal.NewFromInt(97), 2000))
	prices = append(prices, quizPrice("brand-d", "D001", d2, close100, decimal.NewFromInt(103), decimal.NewFromInt(97), 2000))
	// Brand E: 出来高最小 → valueTopN で除外されるべき
	for _, d := range []time.Time{d0, d1, d2} {
		prices = append(prices, quizPrice("brand-e", "E001", d, close100, decimal.NewFromInt(102), decimal.NewFromInt(98), 100))
	}

	entries := SelectQuizUniverse(prices, 3, 2)

	assert.Len(t, entries, 2, "欠損日のあるD・出来高最小のEは除外され、値幅上位2件のみ残る")
	assert.Equal(t, "brand-b", entries[0].StockBrandID, "値幅最大のBが1位")
	assert.Equal(t, 1, entries[0].QuestionOrder)
	assert.Equal(t, "brand-c", entries[1].StockBrandID, "値幅2位のCが2位")
	assert.Equal(t, 2, entries[1].QuestionOrder)

	for _, e := range entries {
		assert.True(t, e.BaseClosePrice.Equal(close100), "基準終値は最終日の終値")
		assert.True(t, e.QuizDate.Equal(d2), "quiz_dateは最終営業日")
	}
}

func TestSelectQuizUniverse_Empty(t *testing.T) {
	assert.Nil(t, SelectQuizUniverse(nil, 600, 300))
}

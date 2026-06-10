package domain_service

import (
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func makePrice(adjClose float64, daysOffset int) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		Date:     time.Now().AddDate(0, 0, daysOffset),
		Adjclose: decimal.NewFromFloat(adjClose),
	}
}

func TestForwardReturns_NoPrices(t *testing.T) {
	_, found := ForwardReturns(nil, models.AnalyzeStockBrandPriceHistoryActionBuy, []int{5})
	assert.False(t, found)
}

func TestForwardReturns_ZeroBasePrice(t *testing.T) {
	prices := []*models.StockBrandDailyPrice{makePrice(0, 0), makePrice(110, 1)}
	_, found := ForwardReturns(prices, models.AnalyzeStockBrandPriceHistoryActionBuy, []int{1})
	assert.False(t, found)
}

func TestForwardReturns_HorizonNotReached(t *testing.T) {
	// prices が2本のみで horizon=5 は未到来
	prices := []*models.StockBrandDailyPrice{makePrice(100, 0), makePrice(110, 1)}
	returns, found := ForwardReturns(prices, models.AnalyzeStockBrandPriceHistoryActionBuy, []int{1, 5})
	assert.True(t, found)
	assert.NotNil(t, returns[1])
	assert.Equal(t, "0.1", returns[1].String())
	assert.Nil(t, returns[5]) // 未到来
}

func TestForwardReturns_SellSignFlip(t *testing.T) {
	// 価格が上昇しても Sell は負リターン
	prices := []*models.StockBrandDailyPrice{makePrice(100, 0), makePrice(110, 1)}
	returns, found := ForwardReturns(prices, models.AnalyzeStockBrandPriceHistoryActionSell, []int{1})
	assert.True(t, found)
	assert.NotNil(t, returns[1])
	assert.True(t, returns[1].IsNegative())
}

func TestForwardReturns_Buy(t *testing.T) {
	prices := make([]*models.StockBrandDailyPrice, 21)
	for i := 0; i < 21; i++ {
		prices[i] = makePrice(float64(100+i), i)
	}
	returns, found := ForwardReturns(prices, models.AnalyzeStockBrandPriceHistoryActionBuy, []int{5, 10, 20})
	assert.True(t, found)
	assert.NotNil(t, returns[5])
	assert.NotNil(t, returns[10])
	assert.NotNil(t, returns[20])
	// P0=100, P5=105 → r5=0.05
	assert.Equal(t, "0.05", returns[5].String())
}

func TestCalcHorizonStats_Empty(t *testing.T) {
	s := calcHorizonStats([]decimal.Decimal{})
	assert.Equal(t, 0, s.EvaluatedCount)
	assert.Equal(t, 0, s.WinCount)
}

func TestCalcHorizonStats_Values(t *testing.T) {
	returns := []decimal.Decimal{
		decimal.NewFromFloat(0.10),
		decimal.NewFromFloat(-0.05),
		decimal.NewFromFloat(0.03),
	}
	s := calcHorizonStats(returns)
	assert.Equal(t, 3, s.EvaluatedCount)
	assert.Equal(t, 2, s.WinCount)
	// 勝率 2/3 ≈ 0.6667
	expected, _ := decimal.NewFromString("0.6667")
	assert.Equal(t, expected, s.WinRate)
}

func TestAggregateSignalPerformance_Groups(t *testing.T) {
	r5a := decimal.NewFromFloat(0.05)
	r5b := decimal.NewFromFloat(-0.02)
	signals := []*models.EvaluatedSignal{
		{Method: "method_a", Returns: map[int]*decimal.Decimal{5: &r5a}},
		{Method: "method_a", Returns: map[int]*decimal.Decimal{5: &r5b}},
		{Method: "method_b", Returns: nil}, // skipped
	}
	summaries := AggregateSignalPerformance(signals, []int{5})
	assert.Len(t, summaries, 2)

	var ma *models.SignalPerformanceSummary
	for _, s := range summaries {
		if s.Method == "method_a" {
			ma = s
		}
	}
	assert.NotNil(t, ma)
	assert.Equal(t, 2, ma.SignalCount)
	assert.Equal(t, 0, ma.SkippedCount)
	assert.Equal(t, 2, ma.Stats[5].EvaluatedCount)
	assert.Equal(t, 1, ma.Stats[5].WinCount)
}

package domain_service

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

// BuildQuizChartSeries クイズのチャート表示用データを組み立てる。
// prices は移動平均のウォームアップのため visibleFrom より十分前（目安75日以上）から渡すこと。
// MA5/25/75 は prices 全体で計算し、visibleFrom 以降の点のみレスポンスに含める。
func BuildQuizChartSeries(prices []*models.StockBrandDailyPrice, visibleFrom time.Time) *models.QuizChart {
	closes := make([]decimal.Decimal, len(prices))
	for i, p := range prices {
		closes[i] = p.Close
	}

	ma5 := smaSeries(closes, 5)
	ma25 := smaSeries(closes, 25)
	ma75 := smaSeries(closes, 75)

	chart := &models.QuizChart{
		Candles: make([]*models.QuizChartCandle, 0, len(prices)),
		MA5:     make([]*models.QuizChartMAPoint, 0, len(prices)),
		MA25:    make([]*models.QuizChartMAPoint, 0, len(prices)),
		MA75:    make([]*models.QuizChartMAPoint, 0, len(prices)),
	}

	for i, p := range prices {
		if p.Date.Before(visibleFrom) {
			continue
		}
		dateStr := p.Date.Format(util.DateLayout)

		chart.Candles = append(chart.Candles, &models.QuizChartCandle{
			Date:   dateStr,
			Open:   p.Open,
			High:   p.High,
			Low:    p.Low,
			Close:  p.Close,
			Volume: p.Volume,
		})
		if !ma5[i].IsZero() {
			chart.MA5 = append(chart.MA5, &models.QuizChartMAPoint{Date: dateStr, Value: ma5[i]})
		}
		if !ma25[i].IsZero() {
			chart.MA25 = append(chart.MA25, &models.QuizChartMAPoint{Date: dateStr, Value: ma25[i]})
		}
		if !ma75[i].IsZero() {
			chart.MA75 = append(chart.MA75, &models.QuizChartMAPoint{Date: dateStr, Value: ma75[i]})
		}
	}

	if len(prices) > 0 {
		chart.QuizDate = prices[len(prices)-1].Date.Format(util.DateLayout)
	}

	return chart
}

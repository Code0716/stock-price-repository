package domain_service

import (
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

// BuildQuizChartSeries クイズのチャート表示用データを組み立てる。
// prices は移動平均のウォームアップのため visibleFrom より十分前（目安75日以上）から渡すこと。
// MA5/25/75 は prices 全体で計算し、visibleFrom 以降の点のみレスポンスに含める。
func BuildQuizChartSeries(prices []*models.StockBrandDailyPrice, visibleFrom time.Time) *models.QuizChart {
	dailyChart := BuildDailyChartSeries(prices, visibleFrom)

	chart := &models.QuizChart{
		Candles: dailyChart.Candles,
		MA5:     dailyChart.MA5,
		MA25:    dailyChart.MA25,
		MA75:    dailyChart.MA75,
	}

	if len(prices) > 0 {
		chart.QuizDate = prices[len(prices)-1].Date.Format(util.DateLayout)
	}

	return chart
}

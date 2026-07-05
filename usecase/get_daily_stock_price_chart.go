package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
)

// dailyChartWarmupMonths MA75のウォームアップに十分な過去データを取得する月数。
// 75営業日 ≒ 約3.7ヶ月 + 休場バッファ。
const dailyChartWarmupMonths = 5

// GetDailyStockPriceChart 指定された銘柄コードと日付範囲に基づいて、日足ローソク足+MA5/25/75のチャートデータを取得します。
func (u *stockBrandsDailyStockPriceInteractorImpl) GetDailyStockPriceChart(ctx context.Context, symbol string, from, to *time.Time) (*models.DailyPriceChart, error) {
	var fetchFrom *time.Time
	var visibleFrom time.Time
	if from != nil {
		warmupFrom := from.AddDate(0, -dailyChartWarmupMonths, 0)
		fetchFrom = &warmupFrom
		visibleFrom = *from
	}

	order := models.SortOrderAsc
	filter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     fetchFrom,
		DateTo:       to,
		DateOrder:    &order,
	}
	prices, err := u.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	return domain_service.BuildDailyChartSeries(prices, visibleFrom), nil
}

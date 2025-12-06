package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// GetDailyStockPrices 指定された銘柄コードと日付範囲に基づいて、日次株価データを取得します。
func (u *stockBrandsDailyStockPriceInteractorImpl) GetDailyStockPrices(ctx context.Context, symbol string, from, to *time.Time) ([]*models.StockBrandDailyPrice, error) {
	// 時系列表示のため昇順でソート
	sortOrder := models.SortOrderAsc
	filter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
		DateOrder:    &sortOrder,
	}
	return u.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, filter)
}

// GetDailyStockPricesWithOrder 指定された銘柄コード、日付範囲、ソート順に基づいて、日次株価データを取得します。
func (u *stockBrandsDailyStockPriceInteractorImpl) GetDailyStockPricesWithOrder(ctx context.Context, symbol string, from, to *time.Time, order *models.SortOrder) ([]*models.StockBrandDailyPrice, error) {
	filter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
		DateOrder:    order,
	}
	return u.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, filter)
}

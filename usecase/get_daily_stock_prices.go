package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// GetDailyStockPrices 指定された銘柄コードと日付範囲に基づいて、日次株価データを取得します。
func (u *stockBrandsDailyStockPriceInteractorImpl) GetDailyStockPrices(ctx context.Context, symbol string, from, to *time.Time) ([]*models.StockBrandDailyPrice, error) {
	filter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
	}
	return u.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, filter)
}

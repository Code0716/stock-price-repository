//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type StockBrandsDailyPriceRepository interface {
	GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error)
	CreateStockBrandDailyPrice(ctx context.Context, dailyPrice []*models.StockBrandDailyPrice) error
	// ListDailyPricesBySymbol symbolから日足を取得する
	ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error)
	// 上場廃止銘柄を削除する。
	DeleteByIDs(ctx context.Context, ids []string) error
}

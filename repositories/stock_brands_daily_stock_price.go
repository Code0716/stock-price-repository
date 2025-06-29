//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type StockBrandsDailyPriceRepository interface {
	GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error)
	CreateStockBrandDailyPrice(ctx context.Context, dailyPrice []*models.StockBrandDailyPrice) error
	// symbols から最新の日足LISTを取得する
	ListLatestPriceBySymbols(ctx context.Context, symbols []*string) ([]*models.StockBrandDailyPrice, error)
	// ListDailyPricesBySymbol symbolから日足を取得する
	ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error)
	// 上場廃止銘柄を削除する。
	DeleteDelisting(ctx context.Context, deleteSymbols []string) error
}

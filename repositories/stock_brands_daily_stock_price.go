//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type StockBrandsDailyPriceRepository interface {
	GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error)
	CreateStockBrandDailyPrice(ctx context.Context, dailyPrice []*models.StockBrandDailyPrice) error
	// ListDailyPricesBySymbol symbolから日足を取得する
	ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error)
	// ListRangePricesBySymbols 複数銘柄の期間中日足を一括取得する（シグナル精度評価用）
	ListRangePricesBySymbols(ctx context.Context, filter models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error)
	// 上場廃止銘柄を削除する。
	DeleteByIDs(ctx context.Context, ids []string) error
	// ListRecentTradingDates onOrBefore以前の直近の営業日（データが存在する日）を新しい順にlimit件取得する（クイズのユニバース選定用）。
	ListRecentTradingDates(ctx context.Context, onOrBefore time.Time, limit int) ([]time.Time, error)
	// ListPricesByDateRange 期間中の全銘柄の日足を取得する（クイズのユニバース選定用）。
	ListPricesByDateRange(ctx context.Context, from, to time.Time) ([]*models.StockBrandDailyPrice, error)
	// FindNextTradingDate afterより後の直近の営業日を1件取得する（存在しなければnil。クイズ採点用）。
	FindNextTradingDate(ctx context.Context, after time.Time) (*time.Time, error)
}

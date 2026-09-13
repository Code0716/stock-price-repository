//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// AdjustedDailyPriceRepository 分割・併合調整済み日足の読み取り専用リポジトリ。
// 実体は stock_brands_daily_price_for_analyze。クイズ・分析・買い候補はこちらを使う。
//
// StockBrandsDailyPriceRepository（素値・取込用）と読み取り6メソッドのシグネチャを
// 完全に同一にしている。呼び出し側(usecase)はフィールドの型宣言を差し替えるだけで
// 移行でき、domain_service を含む本体ロジックは無改修で済む。
//
// 戻り値は StockBrandsDailyPriceRepository と同じ *models.StockBrandDailyPrice。
// for_analyze には stock_brand_id 列が無いため、StockBrandID は基本的に空文字になる。
// ただし ListPricesByDateRange のみ stock_brand を ticker_symbol で JOIN し、
// StockBrandID を正しく埋める（クイズの主要市場フィルタが StockBrandID を使うため）。
type AdjustedDailyPriceRepository interface {
	GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error)
	// ListDailyPricesBySymbol symbolから日足を取得する
	ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error)
	// ListRangePricesBySymbols 複数銘柄の期間中日足を一括取得する（シグナル精度評価用）
	ListRangePricesBySymbols(ctx context.Context, filter models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error)
	// ListRecentTradingDates onOrBefore以前の直近の営業日（データが存在する日）を新しい順にlimit件取得する（クイズのユニバース選定用）。
	ListRecentTradingDates(ctx context.Context, onOrBefore time.Time, limit int) ([]time.Time, error)
	// ListPricesByDateRange 期間中の全銘柄の日足を取得する（クイズのユニバース選定用）。StockBrandIDをstock_brandとのJOINで埋める。
	ListPricesByDateRange(ctx context.Context, from, to time.Time) ([]*models.StockBrandDailyPrice, error)
	// FindNextTradingDate afterより後の直近の営業日を1件取得する（存在しなければnil。クイズ採点用）。
	FindNextTradingDate(ctx context.Context, after time.Time) (*time.Time, error)
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

// StockBrandsDailyPriceForAnalyzeRepository 分析用の日足のリポジトリインターフェース
//
// 分析用日足は3年で自動削除するretention処理を持っていたが、クイズ・分析の読み先として
// 長期間のデータを保持し続ける必要があるため廃止した（DeleteBeforeDateは削除済み）。
// 上場廃止銘柄の削除は DeleteBySymbols で引き続き行う。
type StockBrandsDailyPriceForAnalyzeRepository interface {
	CreateStockBrandDailyPriceForAnalyze(ctx context.Context, dailyPrice []*models.StockBrandDailyPriceForAnalyze) error
	ListLatestPriceBySymbols(ctx context.Context, symbols []*string) ([]*models.StockBrandDailyPriceForAnalyze, error)
	ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPriceForAnalyze, error)
	DeleteBySymbols(ctx context.Context, deleteSymbols []string) error
}

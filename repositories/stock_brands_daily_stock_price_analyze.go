//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

// StockBrandsDailyPriceForAnalyzeRepository 分析用の日足のリポジトリインターフェース
type StockBrandsDailyPriceForAnalyzeRepository interface {
	CreateStockBrandDailyPriceForAnalyze(ctx context.Context, dailyPrice []*models.StockBrandDailyPriceForAnalyze) error
	ListLatestPriceBySymbols(ctx context.Context, symbols []*string) ([]*models.StockBrandDailyPriceForAnalyze, error)
	DeleteBySymbols(ctx context.Context, deleteSymbols []string) error
}

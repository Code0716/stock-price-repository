//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type AnalyzeStockBrandPriceHistoryRepository interface {
	// FindWithFilter 条件に一致する分析履歴を取得する
	FindWithFilter(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) ([]*models.AnalyzeStockBrandPriceHistory, error)
	// DeleteByStockBrandIDs 銘柄IDで一致したものを削除する
	DeleteByStockBrandIDs(ctx context.Context, ids []string) error
}

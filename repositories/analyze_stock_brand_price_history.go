//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type AnalyzeStockBrandPriceHistoryRepository interface {
	// FindWithFilter 条件に一致する分析履歴を取得する
	FindWithFilter(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) ([]*models.AnalyzeStockBrandPriceHistory, error)
	// CountWithFilter 条件に一致する分析履歴の総件数を取得する
	CountWithFilter(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (int64, error)
	// FindDistinctDates フィルタ条件下でデータが存在する日付の一覧を取得する（日付グループモード用）
	FindDistinctDates(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) ([]string, error)
	// CountDistinctDates フィルタ条件下でデータが存在する日付の総数を返す
	CountDistinctDates(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (int64, error)
	// FindByDates 指定した日付リストに含まれる分析履歴を全件取得する
	FindByDates(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter, dates []string) ([]*models.AnalyzeStockBrandPriceHistory, error)
	// FindMultipleSignals 同一日に2つ以上のシグナルが出た銘柄を集計して取得する
	FindMultipleSignals(ctx context.Context, filter *models.MultipleSignalStockFilter) ([]*models.MultipleSignalStock, error)
	// DeleteByStockBrandIDs 銘柄IDで一致したものを削除する
	DeleteByStockBrandIDs(ctx context.Context, ids []string) error
}

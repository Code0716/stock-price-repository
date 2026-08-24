//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type AppliedStockSplitsHistoryRepository interface {
	Exists(ctx context.Context, symbol string, splitDate time.Time) (bool, error)
	Create(ctx context.Context, history *models.AppliedStockSplitHistory) error
	// TruncateAll 全件を物理削除する。5年再構築バッチ(RebuildAnalyzeDailyPrices)専用の破壊的操作。
	TruncateAll(ctx context.Context) error
}

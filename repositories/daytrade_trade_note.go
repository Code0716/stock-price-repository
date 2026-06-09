//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type DaytradeTradeNoteRepository interface {
	// FindByDateRange は期間内のトレード注釈を返す。from / to は nil 可。両方 nil なら全期間。
	FindByDateRange(ctx context.Context, from, to *time.Time) ([]*models.DaytradeTradeNoteRecord, error)
	// Upsert は近似キー（ticker_symbol, executed_on, direction）で upsert する。
	// 全フィールド空の場合（memo="", tags=[], declaredStopPrice=nil）は行を削除する。
	Upsert(ctx context.Context, rec *models.DaytradeTradeNoteRecord) error
}

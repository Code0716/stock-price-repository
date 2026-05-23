//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type DaytradeExecutionRepository interface {
	// 重複は UNIQUE 制約で弾く。返す inserted は実挿入件数。
	BulkInsertIgnore(ctx context.Context, executions []*models.DaytradeExecution) (inserted int, err error)
	// DeleteBySource は指定 source のレコードを全削除し、削除件数を返す。
	DeleteBySource(ctx context.Context, source string) (int64, error)
	// from / to は nil 可。両方 nil なら全期間。
	Aggregate(ctx context.Context, from, to *time.Time, g models.DaytradeSummaryGranularity) ([]*models.DaytradeSummaryBucket, error)
	// AggregateByTickerSymbol 銘柄毎の損益を集計。from / to は nil 可。両方 nil なら全期間。
	AggregateByTickerSymbol(ctx context.Context, from, to *time.Time) ([]*models.DaytradeSymbolSummary, error)
	// 指定日の明細を取得 (executed_on, id ASC)
	FindByDate(ctx context.Context, date time.Time) ([]*models.DaytradeExecution, error)
	// 取り込み済みデータがカバーする期間。データが無ければ (nil, nil, nil)。
	GetCoveredRange(ctx context.Context) (minDate, maxDate *time.Time, err error)
}

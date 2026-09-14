//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type DailyAvoidStockRepository interface {
	// BulkCreate 1日分の避けるべき銘柄をまとめて作成する。
	BulkCreate(ctx context.Context, rows []*models.DailyAvoidStock) error
	// DeleteByAsOfDate 指定日の避けるべき銘柄を全削除する（再実行時の洗い替え用。BulkCreate と同一トランザクションで使う）。
	DeleteByAsOfDate(ctx context.Context, asOfDate time.Time) error
	// ListByAsOfDate 指定日の避けるべき銘柄を avoid_rank 昇順で取得する。
	ListByAsOfDate(ctx context.Context, asOfDate time.Time) ([]*models.DailyAvoidStock, error)
	// ExistsByAsOfDate 指定日の避けるべき銘柄が既に存在するか（バッチの冪等性チェック用）。
	ExistsByAsOfDate(ctx context.Context, asOfDate time.Time) (bool, error)
	// FindLatestAsOfDate 最新の as_of_date を取得する（1件も無ければ nil を返す）。
	FindLatestAsOfDate(ctx context.Context) (*time.Time, error)
	// ListAsOfDates as_of_date を降順に最大 limit 件取得する（日付セレクタ用）。limit<=0 なら無制限。
	ListAsOfDates(ctx context.Context, limit int) ([]time.Time, error)
}

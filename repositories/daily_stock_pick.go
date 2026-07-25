//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type DailyStockPickRepository interface {
	// BulkCreate 1日分の推奨銘柄をまとめて作成する。
	BulkCreate(ctx context.Context, picks []*models.DailyStockPick) error
	// DeleteByPickDate 指定日の推奨を全削除する（再実行時の洗い替え用。BulkCreate と同一トランザクションで使う）。
	DeleteByPickDate(ctx context.Context, pickDate time.Time) error
	// ListByPickDate 指定日の推奨を pick_rank 昇順で取得する。
	ListByPickDate(ctx context.Context, pickDate time.Time) ([]*models.DailyStockPick, error)
	// ExistsByPickDate 指定日の推奨が既に存在するか（バッチの冪等性チェック用）。
	ExistsByPickDate(ctx context.Context, pickDate time.Time) (bool, error)
	// ListPendingEvaluation evaluated_at IS NULL かつ pick_date >= onOrAfter の推奨を pick_date 昇順で取得する。
	ListPendingEvaluation(ctx context.Context, onOrAfter time.Time) ([]*models.DailyStockPick, error)
	// UpdateEvaluations 答え合わせ結果（Return1D/3D/5D, Outcome, EvaluatedAt）を反映する。
	UpdateEvaluations(ctx context.Context, picks []*models.DailyStockPick) error
	// MarkNotified 指定日の推奨に Slack 通知日時を記録する。
	MarkNotified(ctx context.Context, pickDate time.Time, notifiedAt time.Time) error
}

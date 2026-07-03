//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type QuizDailyUniverseRepository interface {
	// BulkCreate 1日分の出題ユニバース（設問）をまとめて作成する。
	BulkCreate(ctx context.Context, entries []*models.QuizUniverseEntry) error
	// ListByQuizDate 指定日の出題ユニバースを出題順で取得する。
	ListByQuizDate(ctx context.Context, quizDate time.Time) ([]*models.QuizUniverseEntry, error)
	// FindLatestQuizDate 最新の出題日を取得する（データが無ければ nil）。
	FindLatestQuizDate(ctx context.Context) (*time.Time, error)
	// FindByQuizDateAndStockBrandID 指定日・銘柄の設問を1件取得する（存在しなければ nil）。
	FindByQuizDateAndStockBrandID(ctx context.Context, quizDate time.Time, stockBrandID string) (*models.QuizUniverseEntry, error)
	// ExistsByQuizDate 指定日の出題ユニバースが既に存在するか（バッチの冪等性チェック用）。
	ExistsByQuizDate(ctx context.Context, quizDate time.Time) (bool, error)
}

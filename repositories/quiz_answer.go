//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// ErrQuizAnswerAlreadyExists 同一 (quiz_date, stock_brand_id) の回答が既に存在する。
var ErrQuizAnswerAlreadyExists = errors.New("quiz answer already exists")

type QuizAnswerRepository interface {
	// Create 回答を1件作成する。同一 (quiz_date, stock_brand_id) が既に存在する場合は ErrQuizAnswerAlreadyExists を返す。
	Create(ctx context.Context, answer *models.QuizAnswer) error
	// ListByQuizDate 指定日の回答一覧を取得する。
	ListByQuizDate(ctx context.Context, quizDate time.Time) ([]*models.QuizAnswer, error)
	// ListByAnsweredDate 指定した回答日（answered_at の日付、JST）の回答一覧を取得する。
	ListByAnsweredDate(ctx context.Context, date time.Time) ([]*models.QuizAnswer, error)
	// ListUngraded 未採点（outcome IS NULL）の回答を全て取得する。
	ListUngraded(ctx context.Context) ([]*models.QuizAnswer, error)
	// UpdateGrading 採点結果（NextClosePrice/ActualReturn/Outcome/Score/GradedAt）を一括反映する。
	UpdateGrading(ctx context.Context, answers []*models.QuizAnswer) error
	// ListAllGraded 統計算出用に採点済みの回答を全て取得する。
	ListAllGraded(ctx context.Context) ([]*models.QuizAnswer, error)
	// ListAll 統計算出用に未採点分を含む全ての回答を answered_at 昇順で取得する。
	ListAll(ctx context.Context) ([]*models.QuizAnswer, error)
}

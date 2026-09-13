//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type NotificationHistoryRepository interface {
	// Create Slack へ送信した通知を1件記録する
	Create(ctx context.Context, notification *models.NotificationHistory) error
	// FindWithFilter 条件に一致する通知履歴を取得する
	FindWithFilter(ctx context.Context, filter *models.NotificationHistoryFilter) ([]*models.NotificationHistory, error)
	// CountWithFilter 条件に一致する通知履歴の総件数を取得する
	CountWithFilter(ctx context.Context, filter *models.NotificationHistoryFilter) (int64, error)
}

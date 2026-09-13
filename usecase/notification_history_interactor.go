//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type notificationHistoryInteractorImpl struct {
	notificationHistoryRepository repositories.NotificationHistoryRepository
}

// NotificationHistoryInteractor GET /notifications のユースケース
type NotificationHistoryInteractor interface {
	GetNotificationHistories(ctx context.Context, filter *models.NotificationHistoryFilter) (*models.PaginatedNotificationHistories, error)
}

// NewNotificationHistoryInteractor コンストラクタ
func NewNotificationHistoryInteractor(
	notificationHistoryRepository repositories.NotificationHistoryRepository,
) NotificationHistoryInteractor {
	return &notificationHistoryInteractorImpl{
		notificationHistoryRepository: notificationHistoryRepository,
	}
}

func (n *notificationHistoryInteractorImpl) GetNotificationHistories(ctx context.Context, filter *models.NotificationHistoryFilter) (*models.PaginatedNotificationHistories, error) {
	if filter == nil {
		filter = &models.NotificationHistoryFilter{}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	filter.Limit = limit
	filter.Page = page

	notifications, err := n.notificationHistoryRepository.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "notificationHistoryRepository.FindWithFilter error")
	}

	total, err := n.notificationHistoryRepository.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "notificationHistoryRepository.CountWithFilter error")
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &models.PaginatedNotificationHistories{
		Notifications: notifications,
		Page:          page,
		Limit:         limit,
		Total:         total,
		TotalPages:    totalPages,
	}, nil
}

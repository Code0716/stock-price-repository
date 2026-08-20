//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type NotificationHistoryRepositoryImpl struct {
	query *genQuery.Query
}

func NewNotificationHistoryRepositoryImpl(db *gorm.DB) repositories.NotificationHistoryRepository {
	return &NotificationHistoryRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (r *NotificationHistoryRepositoryImpl) Create(ctx context.Context, notification *models.NotificationHistory) error {
	tx := TxOrDefault(ctx, r.query)

	id := notification.ID
	if id == "" {
		id = uuid.NewString()
	}

	if err := tx.NotificationHistory.WithContext(ctx).
		Create(r.convertToDBModel(notification, id)); err != nil {
		return errors.Wrap(err, "NotificationHistoryRepositoryImpl.Create error")
	}
	return nil
}

func (r *NotificationHistoryRepositoryImpl) buildWhereQuery(
	q genQuery.INotificationHistoryDo,
	filter *models.NotificationHistoryFilter,
) genQuery.INotificationHistoryDo {
	n := r.query.NotificationHistory

	if filter.ChannelID != "" {
		q = q.Where(n.ChannelID.Eq(filter.ChannelID))
	}
	if filter.From != nil {
		q = q.Where(n.SentAt.Gte(*filter.From))
	}
	if filter.To != nil {
		q = q.Where(n.SentAt.Lte(*filter.To))
	}
	if filter.Query != "" {
		keyword := fmt.Sprintf("%%%s%%", filter.Query)
		q = q.Where(q.Where(n.Title.Like(keyword)).Or(n.Body.Like(keyword)))
	}
	return q
}

func (r *NotificationHistoryRepositoryImpl) FindWithFilter(ctx context.Context, filter *models.NotificationHistoryFilter) ([]*models.NotificationHistory, error) {
	tx := TxOrDefault(ctx, r.query)
	n := tx.NotificationHistory

	q := r.buildWhereQuery(n.WithContext(ctx), filter)

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	rows, err := q.Order(n.SentAt.Desc(), n.ID.Desc()).
		Limit(limit).
		Offset(offset).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "NotificationHistoryRepositoryImpl.FindWithFilter error")
	}

	notifications := make([]*models.NotificationHistory, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, r.convertToDomainModel(row))
	}
	return notifications, nil
}

func (r *NotificationHistoryRepositoryImpl) CountWithFilter(ctx context.Context, filter *models.NotificationHistoryFilter) (int64, error) {
	tx := TxOrDefault(ctx, r.query)
	n := tx.NotificationHistory

	q := r.buildWhereQuery(n.WithContext(ctx), filter)

	count, err := q.Count()
	if err != nil {
		return 0, errors.Wrap(err, "NotificationHistoryRepositoryImpl.CountWithFilter error")
	}
	return count, nil
}

func (r *NotificationHistoryRepositoryImpl) convertToDBModel(m *models.NotificationHistory, id string) *genModel.NotificationHistory {
	return &genModel.NotificationHistory{
		ID:           id,
		Source:       m.Source,
		ChannelID:    m.ChannelID,
		ChannelLabel: m.ChannelLabel,
		Title:        m.Title,
		Body:         m.Body,
		SentAt:       m.SentAt,
	}
}

func (r *NotificationHistoryRepositoryImpl) convertToDomainModel(m *genModel.NotificationHistory) *models.NotificationHistory {
	return &models.NotificationHistory{
		ID:           m.ID,
		Source:       m.Source,
		ChannelID:    m.ChannelID,
		ChannelLabel: m.ChannelLabel,
		Title:        m.Title,
		Body:         m.Body,
		SentAt:       m.SentAt,
		CreatedAt:    m.CreatedAt,
	}
}

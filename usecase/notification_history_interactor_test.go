package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func TestNotificationHistoryInteractor_GetNotificationHistories(t *testing.T) {
	sentAt := time.Date(2026, 8, 20, 17, 45, 0, 0, time.UTC)
	sampleNotifications := []*models.NotificationHistory{
		{
			ID:           "1",
			Source:       models.NotificationHistorySourceStt,
			ChannelID:    "C07RPD4TT5Y",
			ChannelLabel: "株の情報交換",
			Title:        "三角持ち合い銘柄",
			SentAt:       sentAt,
		},
	}

	type fields struct {
		notificationHistoryRepository func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository
	}

	tests := []struct {
		name    string
		fields  fields
		filter  *models.NotificationHistoryFilter
		want    *models.PaginatedNotificationHistories
		wantErr bool
	}{
		{
			name: "正常系: デフォルトのpage/limitが補完される",
			fields: fields{
				notificationHistoryRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.NotificationHistoryFilter{Page: 1, Limit: 50})).
						Return(sampleNotifications, nil)
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(&models.NotificationHistoryFilter{Page: 1, Limit: 50})).
						Return(int64(1), nil)
					return m
				},
			},
			filter: nil,
			want: &models.PaginatedNotificationHistories{
				Notifications: sampleNotifications,
				Page:          1,
				Limit:         50,
				Total:         1,
				TotalPages:    1,
			},
			wantErr: false,
		},
		{
			name: "正常系: TotalPagesが端数を切り上げる",
			fields: fields{
				notificationHistoryRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Any()).Return(sampleNotifications, nil)
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Any()).Return(int64(101), nil)
					return m
				},
			},
			filter: &models.NotificationHistoryFilter{Page: 2, Limit: 50},
			want: &models.PaginatedNotificationHistories{
				Notifications: sampleNotifications,
				Page:          2,
				Limit:         50,
				Total:         101,
				TotalPages:    3,
			},
			wantErr: false,
		},
		{
			name: "異常系: FindWithFilterがエラーを返す",
			fields: fields{
				notificationHistoryRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
			},
			filter:  nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: CountWithFilterがエラーを返す",
			fields: fields{
				notificationHistoryRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Any()).Return(sampleNotifications, nil)
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("db error"))
					return m
				},
			},
			filter:  nil,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			n := &notificationHistoryInteractorImpl{
				notificationHistoryRepository: tt.fields.notificationHistoryRepository(ctrl),
			}

			got, err := n.GetNotificationHistories(context.Background(), tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

package gateway_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func TestRecordingSlackAPIClient_SendMessageByStrings(t *testing.T) {
	type fields struct {
		inner func(ctrl *gomock.Controller) *mock_gateway.MockSlackAPIClientRaw
		repo  func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository
	}

	message := "本文"

	tests := []struct {
		name        string
		fields      fields
		channelName gateway.SlackChannelName
		title       string
		message     *string
		wantTs      string
		wantErr     bool
	}{
		{
			name: "正常系: 株情報系の通知はSlackへ送らずnotification_historyへのみ記録する",
			fields: fields{
				inner: func(ctrl *gomock.Controller) *mock_gateway.MockSlackAPIClientRaw {
					// Slackへは送信しないため inner に EXPECT を張らない
					return mock_gateway.NewMockSlackAPIClientRaw(ctrl)
				},
				repo: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, n *models.NotificationHistory) error {
							assert.Equal(t, models.NotificationHistorySourceSpr, n.Source)
							assert.Equal(t, gateway.SlackChannelNameExchangeStockInfo.String(), n.ChannelID)
							assert.Equal(t, "株の情報交換", n.ChannelLabel)
							assert.Equal(t, "三角持ち合い銘柄", n.Title)
							assert.Equal(t, &message, n.Body)
							return nil
						})
					return m
				},
			},
			channelName: gateway.SlackChannelNameExchangeStockInfo,
			title:       "三角持ち合い銘柄",
			message:     &message,
			wantTs:      "",
			wantErr:     false,
		},
		{
			name: "正常系: dev_notificationはSlackへ送信し記録しない",
			fields: fields{
				inner: func(ctrl *gomock.Controller) *mock_gateway.MockSlackAPIClientRaw {
					m := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
					m.EXPECT().
						SendMessageByStrings(gomock.Any(), gomock.Eq(gateway.SlackChannelNameDevNotification), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("ts", nil)
					return m
				},
				repo: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					// Create が呼ばれないこと自体がアサーション（EXPECTしない）
					return mock_repositories.NewMockNotificationHistoryRepository(ctrl)
				},
			},
			channelName: gateway.SlackChannelNameDevNotification,
			title:       "env: raspberrypi",
			message:     nil,
			wantTs:      "ts",
			wantErr:     false,
		},
		{
			name: "異常系: Slack送信失敗時はエラーを返す",
			fields: fields{
				inner: func(ctrl *gomock.Controller) *mock_gateway.MockSlackAPIClientRaw {
					m := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
					m.EXPECT().
						SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", errors.New("slack error"))
					return m
				},
				repo: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					return mock_repositories.NewMockNotificationHistoryRepository(ctrl)
				},
			},
			channelName: gateway.SlackChannelNameDevNotification,
			title:       "env: raspberrypi",
			message:     &message,
			wantTs:      "",
			wantErr:     true,
		},
		{
			name: "異常系: notification_history記録失敗時はエラーを返す",
			fields: fields{
				inner: func(ctrl *gomock.Controller) *mock_gateway.MockSlackAPIClientRaw {
					// Slackへは送信しないため inner に EXPECT を張らない
					return mock_gateway.NewMockSlackAPIClientRaw(ctrl)
				},
				repo: func(ctrl *gomock.Controller) *mock_repositories.MockNotificationHistoryRepository {
					m := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
					return m
				},
			},
			channelName: gateway.SlackChannelNameExchangeStockInfo,
			title:       "三角持ち合い銘柄",
			message:     &message,
			wantTs:      "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := gateway.NewRecordingSlackAPIClient(tt.fields.inner(ctrl), tt.fields.repo(ctrl), zap.NewNop())

			ts, err := c.SendMessageByStrings(context.Background(), tt.channelName, tt.title, tt.message, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTs, ts)
		})
	}
}

func TestRecordingSlackAPIClient_SendMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inner := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	inner.EXPECT().
		SendMessage(gomock.Any(), gomock.Eq(gateway.SlackChannelNameDevNotification), gomock.Any()).
		Return(nil)

	repo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
	// dev_notification 宛のため Create は呼ばれない

	c := gateway.NewRecordingSlackAPIClient(inner, repo, zap.NewNop())
	err := c.SendMessage(context.Background(), gateway.SlackChannelNameDevNotification, resource.SlackMessageHealthCheck)
	assert.NoError(t, err)
}

func TestRecordingSlackAPIClient_SendBlockMessage(t *testing.T) {
	blockMessage := resource.SlackBlockMessage{
		Text:   "fallback text",
		Blocks: []resource.SlackBlock{resource.NewSlackHeaderBlock("header")},
	}

	t.Run("正常系: dev_notificationはSlackへ送信し記録しない", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		inner := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
		inner.EXPECT().
			SendBlockMessage(gomock.Any(), gomock.Eq(gateway.SlackChannelNameDevNotification), gomock.Eq(blockMessage)).
			Return(nil)

		// dev_notification 宛のため Create は呼ばれない
		repo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)

		c := gateway.NewRecordingSlackAPIClient(inner, repo, zap.NewNop())
		err := c.SendBlockMessage(context.Background(), gateway.SlackChannelNameDevNotification, blockMessage)
		assert.NoError(t, err)
	})

	t.Run("正常系: 株情報系はSlackへ送らずfallback textのみnotification_historyへ記録する", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Slackへは送信しないため inner に EXPECT を張らない
		inner := mock_gateway.NewMockSlackAPIClientRaw(ctrl)

		repo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
		repo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, n *models.NotificationHistory) error {
				assert.Equal(t, blockMessage.Text, n.Title)
				assert.Nil(t, n.Body)
				return nil
			})

		c := gateway.NewRecordingSlackAPIClient(inner, repo, zap.NewNop())
		err := c.SendBlockMessage(context.Background(), gateway.SlackChannelNameExchangeStockInfo, blockMessage)
		assert.NoError(t, err)
	})
}

func TestRecordingSlackAPIClient_SendErrMessageNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inner := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	inner.EXPECT().
		SendErrMessageNotification(gomock.Any(), gomock.Any()).
		Return(nil)

	// エラー通知は記録しないため repo に EXPECT を張らない
	repo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)

	c := gateway.NewRecordingSlackAPIClient(inner, repo, zap.NewNop())
	err := c.SendErrMessageNotification(context.Background(), errors.New("boom"))
	assert.NoError(t, err)
}

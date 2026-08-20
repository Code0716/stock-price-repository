package gateway

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// recordingSlackAPIClient SlackAPIClient をラップし、送信成功時に notification_history へ記録するデコレータ。
// #dev_notification（エラー・実行時間通知）宛は記録対象外。記録に失敗しても Slack 送信の成否には影響させない。
type recordingSlackAPIClient struct {
	inner  SlackAPIClientRaw
	repo   repositories.NotificationHistoryRepository
	logger *zap.Logger
}

// NewRecordingSlackAPIClient inner をラップして通知履歴を記録する SlackAPIClient を返す
func NewRecordingSlackAPIClient(inner SlackAPIClientRaw, repo repositories.NotificationHistoryRepository, logger *zap.Logger) SlackAPIClient {
	return &recordingSlackAPIClient{
		inner:  inner,
		repo:   repo,
		logger: logger,
	}
}

// channelLabel Slack チャンネル定数から front 表示用のラベルへ変換する
func channelLabel(channelName SlackChannelName) string {
	switch channelName {
	case SlackChannelNameExchangeStockInfo:
		return "株の情報交換"
	case SlackChannelNameMachineLearningResult:
		return "機械学習結果"
	case SlackChannelNameBuyingAndSellingNotification:
		return "売買通知"
	case SlackChannelNameDevNotification:
		return "開発通知"
	default:
		return channelName.String()
	}
}

func (c *recordingSlackAPIClient) SendMessage(ctx context.Context, channelName SlackChannelName, message resource.SlackMessage) error {
	if err := c.inner.SendMessage(ctx, channelName, message); err != nil {
		return err
	}
	c.record(ctx, channelName, message.GetMessage(), nil)
	return nil
}

func (c *recordingSlackAPIClient) SendMessageByStrings(ctx context.Context, channelName SlackChannelName, title string, message, ts *string) (string, error) {
	resultTs, err := c.inner.SendMessageByStrings(ctx, channelName, title, message, ts)
	if err != nil {
		return resultTs, err
	}
	c.record(ctx, channelName, title, message)
	return resultTs, nil
}

func (c *recordingSlackAPIClient) SendErrMessageNotification(ctx context.Context, err error) error {
	// dev_notification 宛のため記録しない
	return c.inner.SendErrMessageNotification(ctx, err)
}

func (c *recordingSlackAPIClient) record(ctx context.Context, channelName SlackChannelName, title string, body *string) {
	if channelName == SlackChannelNameDevNotification {
		return
	}

	notification := models.NewNotificationHistory(
		"",
		models.NotificationHistorySourceSpr,
		channelName.String(),
		channelLabel(channelName),
		title,
		body,
		time.Now(),
	)
	if err := c.repo.Create(ctx, notification); err != nil {
		c.logger.Error("recordingSlackAPIClient: failed to record notification history", zap.Error(err))
	}
}

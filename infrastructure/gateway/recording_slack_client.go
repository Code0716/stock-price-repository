package gateway

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// recordingSlackAPIClient SlackAPIClient をラップし、送信先を振り分けるデコレータ。
// #dev_notification（エラー・実行時間通知）宛は従来通り Slack へ送信し notification_history には記録しない。
// それ以外の株情報系通知は Slack へ送信せず、notification_history への記録のみ行う
// （front の /notifications で確認する）。
type recordingSlackAPIClient struct {
	inner  SlackAPIClientRaw
	repo   repositories.NotificationHistoryRepository
	logger *zap.Logger
}

// NewRecordingSlackAPIClient inner をラップして通知の送信先振り分けを行う SlackAPIClient を返す
func NewRecordingSlackAPIClient(inner SlackAPIClientRaw, repo repositories.NotificationHistoryRepository, logger *zap.Logger) SlackAPIClient {
	return &recordingSlackAPIClient{
		inner:  inner,
		repo:   repo,
		logger: logger,
	}
}

// ChannelLabel Slack チャンネル定数から front 表示用のラベルへ変換する
func ChannelLabel(channelName SlackChannelName) string {
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
	if channelName == SlackChannelNameDevNotification {
		return c.inner.SendMessage(ctx, channelName, message)
	}
	return c.record(ctx, channelName, message.GetMessage(), nil)
}

func (c *recordingSlackAPIClient) SendMessageByStrings(ctx context.Context, channelName SlackChannelName, title string, message, ts *string) (string, error) {
	if channelName == SlackChannelNameDevNotification {
		return c.inner.SendMessageByStrings(ctx, channelName, title, message, ts)
	}
	if err := c.record(ctx, channelName, title, message); err != nil {
		return "", err
	}
	return "", nil
}

func (c *recordingSlackAPIClient) SendErrMessageNotification(ctx context.Context, err error) error {
	// dev_notification 宛のため常に Slack へ送信し、記録しない
	return c.inner.SendErrMessageNotification(ctx, err)
}

func (c *recordingSlackAPIClient) SendBlockMessage(ctx context.Context, channelName SlackChannelName, message resource.SlackBlockMessage) error {
	if channelName == SlackChannelNameDevNotification {
		return c.inner.SendBlockMessage(ctx, channelName, message)
	}
	// 現状 dev_notification 以外から呼ばれる想定はないが、将来の誤用で通知が消えないよう
	// フォールバックテキストを notification_history へ記録する（Block Kit の JSON は記録しない）。
	return c.record(ctx, channelName, message.Text, nil)
}

// record notification_history へ記録する。Slack 送信を行わない通知はこの記録が
// 唯一の記録手段になるため、書き込み失敗時は呼び出し元へエラーを返す。
func (c *recordingSlackAPIClient) record(ctx context.Context, channelName SlackChannelName, title string, body *string) error {
	notification := models.NewNotificationHistory(
		"",
		models.NotificationHistorySourceSpr,
		channelName.String(),
		ChannelLabel(channelName),
		title,
		body,
		time.Now(),
	)
	if err := c.repo.Create(ctx, notification); err != nil {
		c.logger.Error("recordingSlackAPIClient: failed to record notification history", zap.Error(err))
		return err
	}
	return nil
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package gateway

import (
	"context"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
)

type SlackChannelName string

const (
	SlackChannelNameExchangeStockInfo            SlackChannelName = "C07RPD4TT5Y" // 株の情報交換チャンネル
	SlackChannelNameMachineLearningResult        SlackChannelName = "C08744MFF7B" // 機械学習結果チャンネル
	SlackChannelNameBuyingAndSellingNotification SlackChannelName = "#buying-and-selling-notification"
	SlackChannelNameDevNotification              SlackChannelName = "#dev_notification"
)

func (s SlackChannelName) String() string {
	return string(s)
}

const (
	SendMessageFindTrendStockByUserLocalMethodTitleRising  string = "ユーザーローカル法銘柄(rising)"
	SendMessageFindTrendStockByUserLocalMethodTitleFalling string = "ユーザーローカル法銘柄(falling)"
)

type SlackAPIClient interface {
	// slackへメッセージを送信する
	SendMessage(ctx context.Context, channelName SlackChannelName, message resource.SlackMessage) error
	SendMessageByStrings(ctx context.Context, channelName SlackChannelName, title string, message, ts *string) (string, error)
	SendErrMessageNotification(ctx context.Context, err error) error
}

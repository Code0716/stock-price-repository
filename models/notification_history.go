package models

import "time"

const (
	NotificationHistorySourceSpr string = "spr"
	NotificationHistorySourceStt string = "stt"
)

// NotificationHistory Slack へ送信した通知の記録。SlackAPIClient のデコレータが送信成功時に書き込む。
type NotificationHistory struct {
	ID           string    `json:"id"`
	Source       string    `json:"source"`
	ChannelID    string    `json:"channelId"`
	ChannelLabel string    `json:"channelLabel"`
	Title        string    `json:"title"`
	Body         *string   `json:"body"`
	SentAt       time.Time `json:"sentAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

func NewNotificationHistory(
	id string,
	source string,
	channelID string,
	channelLabel string,
	title string,
	body *string,
	sentAt time.Time,
) *NotificationHistory {
	return &NotificationHistory{
		ID:           id,
		Source:       source,
		ChannelID:    channelID,
		ChannelLabel: channelLabel,
		Title:        title,
		Body:         body,
		SentAt:       sentAt,
	}
}

// NotificationHistoryFilter GET /notifications の絞り込み条件
type NotificationHistoryFilter struct {
	ChannelID string
	From      *time.Time
	To        *time.Time
	Query     string // title/body の部分一致
	Page      int
	Limit     int
}

type PaginatedNotificationHistories struct {
	Notifications []*NotificationHistory
	Page          int
	Limit         int
	Total         int64
	TotalPages    int
}

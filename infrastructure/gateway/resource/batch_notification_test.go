package resource

import (
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewBatchSuccessMessage(t *testing.T) {
	params := BatchNotificationParams{
		Source:      BatchNotificationSourceSPR,
		Env:         "local",
		CommandName: "create_daily_stock_price_v1",
		Elapsed:     83 * time.Second,
		FinishedAt:  time.Date(2026, 9, 4, 5, 12, 33, 0, time.UTC),
	}

	msg := NewBatchSuccessMessage(params)

	assert.Contains(t, msg.Text, "command name: create_daily_stock_price_v1")
	assert.Contains(t, msg.Text, "time taken:")
	assert.NotContains(t, msg.Text, "<!channel>")

	if assert.Len(t, msg.Blocks, 1) {
		assert.Equal(t, "header", msg.Blocks[0].Type)
	}

	if assert.Len(t, msg.Attachments, 1) {
		att := msg.Attachments[0]
		assert.Equal(t, "#2eb886", att.Color)
		for _, b := range att.Blocks {
			if b.Text != nil {
				assert.NotContains(t, b.Text.Text, "<!channel>")
			}
			for _, f := range b.Fields {
				assert.NotContains(t, f.Text, "<!channel>")
			}
		}
	}
}

func TestNewBatchFailureMessage(t *testing.T) {
	err := errors.Wrap(errors.New("root cause"), "usecase failed")
	params := BatchNotificationParams{
		Source:      BatchNotificationSourceSPR,
		Env:         "prod",
		CommandName: "create_daily_stock_price_v1",
		Elapsed:     83 * time.Second,
		FinishedAt:  time.Date(2026, 9, 4, 5, 12, 33, 0, time.UTC),
		Err:         err,
	}

	msg := NewBatchFailureMessage(params)

	assert.Contains(t, msg.Text, "FAILED")
	if assert.Len(t, msg.Blocks, 2) {
		assert.Equal(t, "header", msg.Blocks[0].Type)
		assert.Equal(t, "section", msg.Blocks[1].Type)
		assert.Contains(t, msg.Blocks[1].Text.Text, "<!channel>")
	}

	if assert.Len(t, msg.Attachments, 1) {
		att := msg.Attachments[0]
		assert.Equal(t, "#d93025", att.Color)

		var errSectionFound bool
		for _, b := range att.Blocks {
			if b.Text != nil && strings.Contains(b.Text.Text, "*エラー*") {
				errSectionFound = true
				assert.Contains(t, b.Text.Text, "```")
				assert.Contains(t, b.Text.Text, "root cause")
			}
		}
		assert.True(t, errSectionFound, "エラー本文を含む section が見つかること")
	}
}

func TestNewBatchFailureMessage_LongError(t *testing.T) {
	// 長大なエラー(コードフェンスを含む)でも全 section が上限内に収まること
	longMsg := strings.Repeat("エラー詳細```破壊テスト```", 500)
	err := errors.New(longMsg)
	params := BatchNotificationParams{
		Source:      BatchNotificationSourceSPR,
		Env:         "prod",
		CommandName: "cmd",
		Elapsed:     time.Second,
		FinishedAt:  time.Now(),
		Err:         err,
	}

	msg := NewBatchFailureMessage(params)

	for _, att := range msg.Attachments {
		for _, b := range att.Blocks {
			if b.Text != nil {
				assert.LessOrEqual(t, len([]rune(b.Text.Text)), SlackSectionTextMaxLen)
			}
		}
	}
}

func TestFormatErrorDetail(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want func(t *testing.T, got string)
	}{
		{
			name: "nilはからで返す",
			err:  nil,
			want: func(t *testing.T, got string) {
				assert.Empty(t, got)
			},
		},
		{
			name: "コードフェンスはエスケープされる",
			err:  errors.New("body```fenced```end"),
			want: func(t *testing.T, got string) {
				assert.NotContains(t, got, "```fenced```")
				assert.Contains(t, got, "'''fenced'''")
			},
		},
		{
			name: "長文は切り詰められ省略メッセージが付く",
			err:  errors.New(strings.Repeat("x", batchErrorDetailMaxLen+100)),
			want: func(t *testing.T, got string) {
				assert.Contains(t, got, "省略")
				assert.LessOrEqual(t, len([]rune(got)), batchErrorDetailMaxLen+len([]rune("\n…（省略。全文はサーバーログを参照）")))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatErrorDetail(tt.err)
			tt.want(t, got)
		})
	}
}

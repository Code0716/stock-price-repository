package resource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthCheckMessage(t *testing.T) {
	params := HealthCheckNotificationParams{
		Source:    BatchNotificationSourceSPR,
		Env:       "local",
		CheckedAt: time.Date(2026, 9, 4, 5, 12, 33, 0, time.UTC),
	}

	msg := NewHealthCheckMessage(params)

	assert.Contains(t, msg.Text, "local")
	assert.Contains(t, msg.Text, "SPR")
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

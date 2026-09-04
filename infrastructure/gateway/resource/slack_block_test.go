package resource

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateForSlack(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		limit int
		want  string
	}{
		{
			name:  "上限未満はそのまま",
			s:     "hello",
			limit: 10,
			want:  "hello",
		},
		{
			name:  "上限ちょうどはそのまま",
			s:     "hello",
			limit: 5,
			want:  "hello",
		},
		{
			name:  "上限超過は末尾に省略記号を付けて切り詰める",
			s:     "hello world",
			limit: 8,
			want:  "hello w…",
		},
		{
			name:  "マルチバイト文字が壊れないこと",
			s:     "日本語のエラーメッセージです",
			limit: 5,
			want:  "日本語の…",
		},
		{
			name:  "空文字はそのまま",
			s:     "",
			limit: 10,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateForSlack(tt.s, tt.limit)
			assert.Equal(t, tt.want, got)
			assert.True(t, len([]rune(got)) <= tt.limit)
		})
	}
}

func TestNewSlackHeaderBlock(t *testing.T) {
	long := strings.Repeat("あ", SlackHeaderTextMaxLen+10)
	block := NewSlackHeaderBlock(long)

	assert.Equal(t, "header", block.Type)
	if assert.NotNil(t, block.Text) {
		assert.Equal(t, "plain_text", block.Text.Type)
		assert.LessOrEqual(t, len([]rune(block.Text.Text)), SlackHeaderTextMaxLen)
		if assert.NotNil(t, block.Text.Emoji) {
			assert.True(t, *block.Text.Emoji)
		}
	}

	b, err := json.Marshal(block)
	assert.NoError(t, err)
	assert.True(t, json.Valid(b))
}

func TestNewSlackFieldsBlock(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		wantLen int
	}{
		{
			name:    "10件以内はすべて保持される",
			fields:  []string{"a", "b", "c"},
			wantLen: 3,
		},
		{
			name: "10件超は切り捨てられる",
			fields: []string{
				"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
			},
			wantLen: SlackFieldsMaxCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewSlackFieldsBlock(tt.fields...)
			assert.Equal(t, "section", block.Type)
			assert.Nil(t, block.Text)
			assert.Len(t, block.Fields, tt.wantLen)
		})
	}
}

func TestSlackBlock_JSONShape(t *testing.T) {
	// section(text) と fields は排他であるべき(omitempty の確認)
	sectionBlock := NewSlackSectionBlock("hello")
	b, err := json.Marshal(sectionBlock)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"type":"section","text":{"type":"mrkdwn","text":"hello"}}`, string(b))

	fieldsBlock := NewSlackFieldsBlock("a", "b")
	b, err = json.Marshal(fieldsBlock)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"type":"section","fields":[{"type":"mrkdwn","text":"a"},{"type":"mrkdwn","text":"b"}]}`, string(b))
}

func TestNewSlackDividerBlock(t *testing.T) {
	block := NewSlackDividerBlock()
	assert.Equal(t, "divider", block.Type)
	assert.Nil(t, block.Text)
	assert.Nil(t, block.Fields)
}

func TestNewSlackContextBlock(t *testing.T) {
	block := NewSlackContextBlock("foo", "bar")
	assert.Equal(t, "context", block.Type)
	assert.Len(t, block.Elements, 2)
	assert.Equal(t, "foo", block.Elements[0].Text)
	assert.Equal(t, "bar", block.Elements[1].Text)
}

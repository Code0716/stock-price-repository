package resource

// SlackBlockMessage は Slack Block Kit 形式のメッセージ。
// Text は blocks が本文としてレンダリングされるため通知バナー・検索・
// アクセシビリティ用のフォールバックとして扱われるが、必須項目として常に設定すること。
type SlackBlockMessage struct {
	Text        string
	Blocks      []SlackBlock
	Attachments []SlackAttachment
	ThreadTS    *string
}

// SlackBlock は Block Kit の1ブロックを表す。用途に応じてフィールドの一部のみ使用する。
type SlackBlock struct {
	Type     string            `json:"type"`
	Text     *SlackTextObject  `json:"text,omitempty"`
	Fields   []SlackTextObject `json:"fields,omitempty"`
	Elements []SlackTextObject `json:"elements,omitempty"`
}

// SlackTextObject は Block Kit の text object。
type SlackTextObject struct {
	Type  string `json:"type"` // "mrkdwn" | "plain_text"
	Text  string `json:"text"`
	Emoji *bool  `json:"emoji,omitempty"` // plain_text のみ有効
}

// SlackAttachment は色帯付きの補助ブロック群。
type SlackAttachment struct {
	Color    string       `json:"color,omitempty"`
	Fallback string       `json:"fallback,omitempty"`
	Blocks   []SlackBlock `json:"blocks,omitempty"`
}

// Slack Block Kit の文字数・件数上限。
const (
	SlackHeaderTextMaxLen  = 150
	SlackSectionTextMaxLen = 3000
	SlackFieldTextMaxLen   = 2000
	SlackFieldsMaxCount    = 10
)

func mrkdwnText(text string) *SlackTextObject {
	return &SlackTextObject{Type: "mrkdwn", Text: text}
}

// NewSlackHeaderBlock は header ブロックを生成する。plain_text のみ許可され、150文字上限。
func NewSlackHeaderBlock(text string) SlackBlock {
	emoji := true
	return SlackBlock{
		Type: "header",
		Text: &SlackTextObject{
			Type:  "plain_text",
			Text:  TruncateForSlack(text, SlackHeaderTextMaxLen),
			Emoji: &emoji,
		},
	}
}

// NewSlackSectionBlock は mrkdwn の section ブロックを生成する。3000文字上限。
func NewSlackSectionBlock(mrkdwn string) SlackBlock {
	return SlackBlock{
		Type: "section",
		Text: mrkdwnText(TruncateForSlack(mrkdwn, SlackSectionTextMaxLen)),
	}
}

// NewSlackFieldsBlock は複数の mrkdwn フィールドを持つ section ブロックを生成する。
// Slack の仕様上 10 件を超えるフィールドは切り捨てる。
func NewSlackFieldsBlock(fields ...string) SlackBlock {
	if len(fields) > SlackFieldsMaxCount {
		fields = fields[:SlackFieldsMaxCount]
	}
	objs := make([]SlackTextObject, 0, len(fields))
	for _, f := range fields {
		objs = append(objs, *mrkdwnText(TruncateForSlack(f, SlackFieldTextMaxLen)))
	}
	return SlackBlock{
		Type:   "section",
		Fields: objs,
	}
}

// NewSlackDividerBlock は区切り線ブロックを生成する。
func NewSlackDividerBlock() SlackBlock {
	return SlackBlock{Type: "divider"}
}

// NewSlackContextBlock は補足情報用の context ブロックを生成する。
func NewSlackContextBlock(mrkdwn ...string) SlackBlock {
	elements := make([]SlackTextObject, 0, len(mrkdwn))
	for _, m := range mrkdwn {
		elements = append(elements, *mrkdwnText(TruncateForSlack(m, SlackSectionTextMaxLen)))
	}
	return SlackBlock{
		Type:     "context",
		Elements: elements,
	}
}

// TruncateForSlack は rune 単位で文字列を limit 文字まで切り詰め、
// 切り詰めが発生した場合は末尾に省略記号を付与する。マルチバイト文字を破壊しない。
func TruncateForSlack(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	const ellipsis = "…"
	ellipsisLen := len([]rune(ellipsis))
	if limit <= ellipsisLen {
		return string(r[:limit])
	}
	return string(r[:limit-ellipsisLen]) + ellipsis
}

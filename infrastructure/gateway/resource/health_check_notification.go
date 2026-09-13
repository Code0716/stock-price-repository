package resource

import (
	"fmt"
	"time"
)

// HealthCheckNotificationParams はヘルスチェック通知に必要な情報。
// config / models を import せず、呼び出し元がプリミティブ値を渡す設計にすることで
// resource パッケージのドメイン非依存性を保つ。
type HealthCheckNotificationParams struct {
	Source    BatchNotificationSource
	Env       string
	CheckedAt time.Time
}

// NewHealthCheckMessage は動作確認用ヘルスチェックの Block Kit メッセージを生成する。
// バッチ成功通知と同じく定期実行のノイズを避けるため @channel は付けない。
func NewHealthCheckMessage(p HealthCheckNotificationParams) SlackBlockMessage {
	fallback := fmt.Sprintf("💚 [%s] health check ok (%s)", p.Env, p.Source)

	return SlackBlockMessage{
		Text: fallback,
		Blocks: []SlackBlock{
			NewSlackHeaderBlock(":heartbeat: ヘルスチェック"),
		},
		Attachments: []SlackAttachment{
			{
				Color:    "#2eb886",
				Fallback: fallback,
				Blocks: []SlackBlock{
					NewSlackFieldsBlock(
						"*結果*\n:white_check_mark: OK",
						fmt.Sprintf("*環境*\n`%s`", p.Env),
						fmt.Sprintf("*実行時刻*\n%s", p.CheckedAt.Format("2006-01-02 15:04:05")),
					),
					NewSlackContextBlock(fmt.Sprintf("%s CLI batch ｜ 疎通確認", p.Source)),
				},
			},
		},
	}
}

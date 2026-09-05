package resource

import (
	"fmt"
	"strings"
	"time"
)

// BatchNotificationSource はバッチ通知の発信元。SPR/STT どちらの CLI バッチかを表す。
// STT は入力プリミティブのみで完結する設計にしているため、値だけ先に用意しておく。
type BatchNotificationSource string

const (
	BatchNotificationSourceSPR BatchNotificationSource = "SPR"
	BatchNotificationSourceSTT BatchNotificationSource = "STT"
)

// batchErrorDetailMaxLen は section ブロック上限(3000)からラベル・コードフェンス・
// 省略記号のマージンを引いた、エラー本文に割り当てる上限文字数。
const batchErrorDetailMaxLen = 2800

// BatchNotificationParams は CLI バッチの実行結果通知に必要な情報。
// config / models を import せず、呼び出し元がプリミティブ値を渡す設計にすることで
// resource パッケージのドメイン非依存性を保つ。
type BatchNotificationParams struct {
	Source      BatchNotificationSource
	Env         string
	CommandName string
	Elapsed     time.Duration
	FinishedAt  time.Time
	Err         error // 失敗時のみ設定
}

// NewBatchSuccessMessage はバッチ成功時の Block Kit メッセージを生成する。
// チャンネル全体への通知(@channel)は行わず、緑帯1枚で簡潔に示す。
func NewBatchSuccessMessage(p BatchNotificationParams) SlackBlockMessage {
	elapsed := p.Elapsed.Round(time.Millisecond)
	fallback := fmt.Sprintf(
		"✅ [%s] command name: %s succeeded | time taken: %s",
		p.Env, p.CommandName, elapsed,
	)

	return SlackBlockMessage{
		Text: fallback,
		Blocks: []SlackBlock{
			NewSlackHeaderBlock(":white_check_mark: バッチ成功"),
		},
		Attachments: []SlackAttachment{
			{
				Color:    "#2eb886",
				Fallback: fallback,
				Blocks: []SlackBlock{
					NewSlackFieldsBlock(
						fmt.Sprintf("*コマンド*\n`%s`", p.CommandName),
						fmt.Sprintf("*環境*\n`%s`", p.Env),
						fmt.Sprintf("*所要時間*\n%s", elapsed),
						fmt.Sprintf("*完了時刻*\n%s", p.FinishedAt.Format("2006-01-02 15:04:05")),
					),
					NewSlackContextBlock(fmt.Sprintf("%s CLI batch", p.Source)),
				},
			},
		},
	}
}

// NewBatchFailureMessage はバッチ失敗時の Block Kit メッセージを生成する。
// 赤帯・@channel メンション・整形済みエラー本文で目立たせる。
func NewBatchFailureMessage(p BatchNotificationParams) SlackBlockMessage {
	elapsed := p.Elapsed.Round(time.Millisecond)
	errMsg := ""
	if p.Err != nil {
		errMsg = p.Err.Error()
	}
	fallback := fmt.Sprintf(
		"🚨 [%s] command name: %s FAILED | time taken: %s | %s",
		p.Env, p.CommandName, elapsed, TruncateForSlack(errMsg, SlackHeaderTextMaxLen),
	)

	return SlackBlockMessage{
		Text: fallback,
		Blocks: []SlackBlock{
			NewSlackHeaderBlock(":rotating_light: バッチ失敗"),
			NewSlackSectionBlock(fmt.Sprintf("<!channel> `%s` が失敗しました。", p.CommandName)),
		},
		Attachments: []SlackAttachment{
			{
				Color:    "#d93025",
				Fallback: fallback,
				Blocks: []SlackBlock{
					NewSlackFieldsBlock(
						fmt.Sprintf("*コマンド*\n`%s`", p.CommandName),
						fmt.Sprintf("*環境*\n`%s`", p.Env),
						fmt.Sprintf("*所要時間*\n%s", elapsed),
						fmt.Sprintf("*発生時刻*\n%s", p.FinishedAt.Format("2006-01-02 15:04:05")),
					),
					NewSlackDividerBlock(),
					NewSlackSectionBlock(fmt.Sprintf("*エラー*\n```%s```", formatErrorDetail(p.Err))),
					NewSlackContextBlock(fmt.Sprintf("%s CLI batch ｜ 全文はサーバーログを参照", p.Source)),
				},
			},
		},
	}
}

// formatErrorDetail はエラーをコードフェンス表示向けに整形する。
// スタックトレース込みの全文を取得したうえで、コードフェンス破壊を防ぐエスケープと
// rune 単位の truncate を行う。全文は呼び出し元が別途ログへ出力する前提。
func formatErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	detail := fmt.Sprintf("%+v", err)
	detail = strings.ReplaceAll(detail, "```", "'''")
	if truncated := TruncateForSlack(detail, batchErrorDetailMaxLen); truncated != detail {
		return truncated + "\n…（省略。全文はサーバーログを参照）"
	}
	return detail
}

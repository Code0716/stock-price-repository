//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package gateway

import "context"

// EconomicCalendarClient 日銀・FOMC・米雇用統計/CPI の公開スケジュールを取得するクライアント。
// 生の bytes を返すだけで、パースは usecase/marketcal の純関数が担う。
type EconomicCalendarClient interface {
	// FetchBOJSchedulePage 日本銀行「金融政策決定会合」の日程ページ(HTML)を取得する。
	FetchBOJSchedulePage(ctx context.Context) ([]byte, error)
	// FetchFOMCPage FRB の FOMC 会合カレンダーページ(HTML)を取得する。
	FetchFOMCPage(ctx context.Context) ([]byte, error)
	// FetchBLSICS 米労働統計局(BLS)の公式 iCalendar を取得する。
	FetchBLSICS(ctx context.Context) ([]byte, error)
}

package models

import "time"

// MarketHolDiv J-Quants 取引カレンダーの休日区分（HolDiv）。
const (
	MarketHolDivClosed        = 0 // 休場
	MarketHolDivOpen          = 1 // 営業
	MarketHolDivHalfDay       = 2 // 半日取引
	MarketHolDivHolidayTrades = 3 // 休場だが祝日取引あり
)

// MarketEventKind 市場イベントの種別。
const (
	MarketEventKindSQMajor = "sq_major"
	MarketEventKindSQMini  = "sq_mini"
	MarketEventKindBOJ     = "boj"
	MarketEventKindFOMC    = "fomc"
	MarketEventKindUSCPI   = "us_cpi"
	MarketEventKindUSNFP   = "us_nfp"
)

// MarketEventSource 市場イベントの取得元。
const (
	MarketEventSourceJQuantsCalc = "jquants_calc"
	MarketEventSourceBOJHTML     = "boj_html"
	MarketEventSourceFedHTML     = "fed_html"
	MarketEventSourceBLSICS      = "bls_ics"
	MarketEventSourceManual      = "manual"
)

// MarketCalendarDay 取引カレンダーの1日分。
type MarketCalendarDay struct {
	Date   time.Time `json:"date"`
	HolDiv int       `json:"holDiv"`
}

// IsTradingDay 取引が行われる日（営業・半日取引）かどうか。
func (d *MarketCalendarDay) IsTradingDay() bool {
	return d.HolDiv == MarketHolDivOpen || d.HolDiv == MarketHolDivHalfDay
}

// MarketEvent デイトレカレンダーに表示する市場イベント。
type MarketEvent struct {
	Date   time.Time `json:"date"`
	Kind   string    `json:"kind"`
	Label  string    `json:"label"`
	Source string    `json:"-"`
}

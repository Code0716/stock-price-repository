// Package marketcal 市場イベント（SQ・日銀会合・FOMC・米CPI・米雇用統計）の取得結果を解釈する純関数群。
// I/O は持たない。取得は gateway、保存は usecase が担う。
package marketcal

import (
	"sort"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// expectedMeetingsPerYear 日銀・FOMC の年間定例会合数。これと異なる年はパース異常として扱う。
const expectedMeetingsPerYear = 8

// 表示ラベル。front のチップ表示に使う。
const (
	labelSQMajor = "メジャーSQ"
	labelSQMini  = "SQ"
	labelBOJ     = "日銀"
	labelFOMC    = "FOMC結果"
	labelUSCPI   = "米CPI"
	labelUSNFP   = "米雇用統計"
)

func newEvent(date time.Time, kind, label, source string) *models.MarketEvent {
	return &models.MarketEvent{
		Date:   time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local),
		Kind:   kind,
		Label:  label,
		Source: source,
	}
}

// sortEvents 日付昇順（同日は kind 昇順）に並べる。
func sortEvents(events []*models.MarketEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].Date.Equal(events[j].Date) {
			return events[i].Date.Before(events[j].Date)
		}
		return events[i].Kind < events[j].Kind
	})
}

// countByYear 年ごとの件数を数える。
func countByYear(events []*models.MarketEvent) map[int]int {
	m := map[int]int{}
	for _, e := range events {
		m[e.Date.Year()]++
	}
	return m
}

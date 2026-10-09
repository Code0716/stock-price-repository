package marketcal

import (
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// CalcSQ 取引カレンダーから from〜to（両端含む、月単位）の SQ 日を計算する。
// 毎月の第2金曜が SQ 日で、その日が休場なら直前の営業日に繰り上げる。3/6/9/12月はメジャーSQ。
// カレンダーに第2金曜が載っていない月は判定できないためスキップする。
func CalcSQ(days []*models.MarketCalendarDay, from, to time.Time) []*models.MarketEvent {
	byDate := make(map[string]*models.MarketCalendarDay, len(days))
	for _, d := range days {
		byDate[dateKey(d.Date)] = d
	}

	var events []*models.MarketEvent
	cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.Local)
	last := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.Local)
	for !cur.After(last) {
		if sq, ok := sqDay(byDate, cur.Year(), cur.Month()); ok && !sq.Before(startOfDay(from)) && !sq.After(startOfDay(to)) {
			kind, label := models.MarketEventKindSQMini, labelSQMini
			if m := cur.Month(); m == time.March || m == time.June || m == time.September || m == time.December {
				kind, label = models.MarketEventKindSQMajor, labelSQMajor
			}
			events = append(events, newEvent(sq, kind, label, models.MarketEventSourceJQuantsCalc))
		}
		cur = cur.AddDate(0, 1, 0)
	}
	return events
}

// sqDay その月の SQ 日を返す。第2金曜の営業日判定ができない場合は ok=false。
func sqDay(byDate map[string]*models.MarketCalendarDay, year int, month time.Month) (time.Time, bool) {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	offset := (int(time.Friday) - int(first.Weekday()) + 7) % 7
	secondFriday := first.AddDate(0, 0, offset+7)

	d := secondFriday
	// 連休が長くても数日以内に営業日があるので、10日で打ち切る。
	for i := 0; i < 10; i++ {
		day, ok := byDate[dateKey(d)]
		if !ok {
			return time.Time{}, false
		}
		if day.IsTradingDay() {
			return d, true
		}
		d = d.AddDate(0, 0, -1)
	}
	return time.Time{}, false
}

func dateKey(t time.Time) string {
	return t.Format("2006-01-02")
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

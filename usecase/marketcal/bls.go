package marketcal

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
)

var dtstartRe = regexp.MustCompile(`^DTSTART[^:]*:(\d{8})`)

// ParseBLSICS BLS 公式 iCalendar から米CPI・米雇用統計の発表日を取り出す。
// 発表は米東部時間 8:30（日本時間 21:30/22:30）なので、日本時間でも同じ暦日になる。DTSTART の日付部分をそのまま使う。
// 地域別 CPI（"Consumer Price Index, Pacific Region" 等）や州別雇用統計は対象外。
func ParseBLSICS(data []byte) ([]*models.MarketEvent, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	// RFC 5545 の行の折り返し（改行＋空白/タブ）を展開する。
	text = regexp.MustCompile(`\n[ \t]`).ReplaceAllString(text, "")

	var events []*models.MarketEvent
	seen := map[string]bool{}
	var inEvent bool
	var dtstart, summary string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "BEGIN:VEVENT":
			inEvent, dtstart, summary = true, "", ""
		case line == "END:VEVENT":
			if inEvent {
				if e, err := blsEvent(dtstart, summary); err != nil {
					return nil, err
				} else if e != nil && !seen[dateKey(e.Date)+e.Kind] {
					seen[dateKey(e.Date)+e.Kind] = true
					events = append(events, e)
				}
			}
			inEvent = false
		case inEvent && strings.HasPrefix(line, "DTSTART"):
			if m := dtstartRe.FindStringSubmatch(line); m != nil {
				dtstart = m[1]
			}
		case inEvent && strings.HasPrefix(line, "SUMMARY:"):
			summary = strings.TrimPrefix(line, "SUMMARY:")
		}
	}
	if len(events) == 0 {
		return nil, errors.New("ParseBLSICS: CPI/雇用統計のイベントが1件も見つかりません")
	}
	sortEvents(events)
	return events, nil
}

func blsEvent(dtstart, summary string) (*models.MarketEvent, error) {
	kind, label := classifyBLSSummary(summary)
	if kind == "" {
		return nil, nil
	}
	if dtstart == "" {
		return nil, errors.Errorf("ParseBLSICS: DTSTART がありません summary=%q", summary)
	}
	date, err := time.ParseInLocation("20060102", dtstart, time.Local)
	if err != nil {
		return nil, errors.Wrap(err, "ParseBLSICS: DTSTART parse error")
	}
	return newEvent(date, kind, label, models.MarketEventSourceBLSICS), nil
}

// classifyBLSSummary SUMMARY から指標を判定する。実物の ics は "Consumer Price Index" / "Employment Situation" の
// 完全一致。"Employment Situation of Veterans" や地域別 CPI など別の指標を拾わないよう、
// 完全一致か「<指標名> for <対象月>」の形だけを対象にする。
func classifyBLSSummary(summary string) (kind, label string) {
	s := strings.TrimSpace(summary)
	switch {
	case isBLSIndicator(s, "Consumer Price Index"):
		return models.MarketEventKindUSCPI, labelUSCPI
	case isBLSIndicator(s, "Employment Situation"):
		return models.MarketEventKindUSNFP, labelUSNFP
	}
	return "", ""
}

func isBLSIndicator(summary, name string) bool {
	return summary == name || strings.HasPrefix(summary, name+" for ")
}

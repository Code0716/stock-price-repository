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

func classifyBLSSummary(summary string) (kind, label string) {
	s := strings.TrimSpace(summary)
	switch {
	case strings.HasPrefix(s, "Consumer Price Index") && !strings.Contains(s, "Region") && !strings.Contains(s, "Area") && !strings.Contains(s, "Metropolitan"):
		return models.MarketEventKindUSCPI, labelUSCPI
	case strings.HasPrefix(s, "Employment Situation"):
		return models.MarketEventKindUSNFP, labelUSNFP
	}
	return "", ""
}

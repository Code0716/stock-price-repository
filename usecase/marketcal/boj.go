package marketcal

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/html"

	"github.com/Code0716/stock-price-repository/models"
)

var (
	bojYearRe = regexp.MustCompile(`^\s*(\d{4})年\s*$`)
	// 「1月22日（木）・23日（金）」「12月17日（木）・18日（金）」「3月18日（水）」（1日開催）に対応する。
	bojMeetingRe = regexp.MustCompile(`(\d{1,2})月\s*(\d{1,2})日（.）(?:・(?:(\d{1,2})月\s*)?(\d{1,2})日（.）)?`)
)

// ParseBOJ 日銀「金融政策決定会合」日程ページから会合日（最終日＝結果公表日）を取り出す。
// 「YYYY年」の h2 見出しで年を決め、その下の表の各行の1列目から日付を読む。
func ParseBOJ(data []byte) ([]*models.MarketEvent, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "ParseBOJ: html.Parse error")
	}

	var events []*models.MarketEvent
	var walkErr error
	year := 0
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if walkErr != nil {
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2":
				year = 0
				if m := bojYearRe.FindStringSubmatch(nodeText(n)); m != nil {
					year, _ = strconv.Atoi(m[1])
				}
			case "tr":
				if year != 0 {
					if td := firstChildElement(n, "td"); td != nil {
						e, err := bojMeetingEvent(year, nodeText(td))
						if err != nil {
							walkErr = err
							return
						}
						if e != nil {
							events = append(events, e)
						}
					}
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if walkErr != nil {
		return nil, walkErr
	}
	if len(events) == 0 {
		return nil, errors.New("ParseBOJ: 会合日が1件も見つかりません")
	}
	sortEvents(events)
	return events, nil
}

func bojMeetingEvent(year int, cell string) (*models.MarketEvent, error) {
	m := bojMeetingRe.FindStringSubmatch(cell)
	if m == nil {
		return nil, nil
	}
	month, _ := strconv.Atoi(m[1])
	day, _ := strconv.Atoi(m[2])
	if m[4] != "" { // 2日開催: 最終日を使う
		day, _ = strconv.Atoi(m[4])
		if m[3] != "" {
			month, _ = strconv.Atoi(m[3])
		}
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if date.Month() != time.Month(month) || date.Day() != day {
		return nil, errors.Errorf("ParseBOJ: 不正な日付 %d年%d月%d日", year, month, day)
	}
	return newEvent(date, models.MarketEventKindBOJ, labelBOJ, models.MarketEventSourceBOJHTML), nil
}

func firstChildElement(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

// nodeText 配下のテキストを連結して返す。
func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

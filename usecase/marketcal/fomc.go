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
	fomcYearRe  = regexp.MustCompile(`^(\d{4}) FOMC Meetings$`)
	fomcMonthRe = regexp.MustCompile(`^(January|February|March|April|May|June|July|August|September|October|November|December)(?:/(January|February|March|April|May|June|July|August|September|October|November|December))?$`)
	// 「27-28」「17-18*」（* は経済見通し付き会合）。月またぎは「31-1」。
	fomcDateRe = regexp.MustCompile(`^(\d{1,2})(?:-(\d{1,2}))?\*?$`)
)

var monthByName = map[string]time.Month{
	"January": time.January, "February": time.February, "March": time.March, "April": time.April,
	"May": time.May, "June": time.June, "July": time.July, "August": time.August,
	"September": time.September, "October": time.October, "November": time.November, "December": time.December,
}

// ParseFOMC FRB の FOMC 会合カレンダーページから会合日程を取り出す。
// 「YYYY FOMC Meetings」の見出し、続く「月」「日」のテキストを順に読む。
// 表示日は、結果発表（会合2日目の米国時間午後）の日本時間の日付＝2日目の翌日とする。
func ParseFOMC(data []byte) ([]*models.MarketEvent, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "ParseFOMC: html.Parse error")
	}

	var s fomcScanner
	for _, tok := range textTokens(doc) {
		if err := s.scan(tok); err != nil {
			return nil, err
		}
	}
	if len(s.events) == 0 {
		return nil, errors.New("ParseFOMC: 会合日が1件も見つかりません")
	}
	sortEvents(s.events)
	return s.events, nil
}

// fomcScanner テキストトークンを先頭から順に読み、年見出し→月→日の並びから会合を拾う状態機械。
type fomcScanner struct {
	year       int
	startMonth time.Month // 0 は「月のトークン待ち」
	endMonth   time.Month
	events     []*models.MarketEvent
}

func (s *fomcScanner) scan(tok string) error {
	if m := fomcYearRe.FindStringSubmatch(tok); m != nil {
		s.year, _ = strconv.Atoi(m[1])
		s.startMonth, s.endMonth = 0, 0
		return nil
	}
	if s.year == 0 {
		return nil
	}
	if m := fomcMonthRe.FindStringSubmatch(tok); m != nil {
		s.startMonth = monthByName[m[1]]
		s.endMonth = s.startMonth
		if m[2] != "" {
			s.endMonth = monthByName[m[2]]
		}
		return nil
	}
	if s.startMonth == 0 {
		return nil
	}
	m := fomcDateRe.FindStringSubmatch(tok)
	if m == nil {
		return nil
	}
	return s.addMeeting(tok, m)
}

func (s *fomcScanner) addMeeting(tok string, m []string) error {
	endDay, _ := strconv.Atoi(m[1])
	if m[2] != "" {
		endDay, _ = strconv.Atoi(m[2])
	}
	endDate := time.Date(s.year, s.endMonth, endDay, 0, 0, 0, 0, time.Local)
	if endDate.Month() != s.endMonth || endDate.Day() != endDay {
		return errors.Errorf("ParseFOMC: 不正な日付 %d %s %s", s.year, s.endMonth, tok)
	}
	// 日本時間の日付 = 会合2日目の翌日
	s.events = append(s.events, newEvent(endDate.AddDate(0, 0, 1), models.MarketEventKindFOMC, labelFOMC, models.MarketEventSourceFedHTML))
	s.startMonth, s.endMonth = 0, 0
	return nil
}

// textTokens ドキュメント順に、空白を畳んだ非空テキストを集める（script/style は除く）。
func textTokens(root *html.Node) []string {
	var tokens []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			if t := strings.Join(strings.Fields(n.Data), " "); t != "" {
				tokens = append(tokens, t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return tokens
}

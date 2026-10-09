package marketcal

import (
	"sort"
	"time"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"

	"github.com/Code0716/stock-price-repository/models"
)

// manualEventYAML 手入力 YAML の1行。
type manualEventYAML struct {
	Date  string `yaml:"date"`
	Kind  string `yaml:"kind"`
	Label string `yaml:"label"`
}

var validKinds = map[string]bool{
	models.MarketEventKindSQMajor: true,
	models.MarketEventKindSQMini:  true,
	models.MarketEventKindBOJ:     true,
	models.MarketEventKindFOMC:    true,
	models.MarketEventKindUSCPI:   true,
	models.MarketEventKindUSNFP:   true,
}

// ParseManualEvents 手入力 YAML（{date, kind, label} の配列）をイベントに変換する。
// スクレイピング元が壊れたときの復旧用で、(kind, 年) の組に1件でもあれば、その組は手入力で置き換える。
func ParseManualEvents(data []byte) ([]*models.MarketEvent, error) {
	var rows []manualEventYAML
	if err := yaml.Unmarshal(data, &rows); err != nil {
		return nil, errors.Wrap(err, "ParseManualEvents: yaml.Unmarshal error")
	}

	events := make([]*models.MarketEvent, 0, len(rows))
	for i, r := range rows {
		if !validKinds[r.Kind] {
			return nil, errors.Errorf("ParseManualEvents: %d件目の kind が不正です: %q", i+1, r.Kind)
		}
		date, err := time.ParseInLocation("2006-01-02", r.Date, time.Local)
		if err != nil {
			return nil, errors.Wrapf(err, "ParseManualEvents: %d件目の date が不正です: %q", i+1, r.Date)
		}
		label := r.Label
		if label == "" {
			label = defaultLabel(r.Kind)
		}
		events = append(events, newEvent(date, r.Kind, label, models.MarketEventSourceManual))
	}
	sortEvents(events)
	return events, nil
}

func defaultLabel(kind string) string {
	switch kind {
	case models.MarketEventKindSQMajor:
		return labelSQMajor
	case models.MarketEventKindSQMini:
		return labelSQMini
	case models.MarketEventKindBOJ:
		return labelBOJ
	case models.MarketEventKindFOMC:
		return labelFOMC
	case models.MarketEventKindUSCPI:
		return labelUSCPI
	case models.MarketEventKindUSNFP:
		return labelUSNFP
	}
	return kind
}

// ValidateAnnualMeetings 日銀・FOMC の定例会合が、fromYear〜toYear のうち取得できた年それぞれで
// 年8回ちょうどであることを確かめ、その年のイベントだけを返す。
// 取得できなかった年（FOMC が翌年分を未公表など）はスキップする。1年も無ければエラー。
func ValidateAnnualMeetings(events []*models.MarketEvent, fromYear, toYear int) ([]*models.MarketEvent, error) {
	counts := countByYear(events)
	var out []*models.MarketEvent
	covered := 0
	for y := fromYear; y <= toYear; y++ {
		n, ok := counts[y]
		if !ok {
			continue
		}
		if n != expectedMeetingsPerYear {
			return nil, errors.Errorf("ValidateAnnualMeetings: %d年の会合数が %d 件です（期待値 %d 件）。ページ構造の変更の可能性があります", y, n, expectedMeetingsPerYear)
		}
		covered++
	}
	if covered == 0 {
		return nil, errors.Errorf("ValidateAnnualMeetings: %d〜%d年の会合が見つかりません", fromYear, toYear)
	}
	for _, e := range events {
		if e.Date.Year() >= fromYear && e.Date.Year() <= toYear {
			out = append(out, e)
		}
	}
	return out, nil
}

// FilterYears fromYear〜toYear の範囲のイベントだけを返す。
func FilterYears(events []*models.MarketEvent, fromYear, toYear int) []*models.MarketEvent {
	var out []*models.MarketEvent
	for _, e := range events {
		if e.Date.Year() >= fromYear && e.Date.Year() <= toYear {
			out = append(out, e)
		}
	}
	return out
}

// KindYear 洗い替えの単位（イベント種別 × 年）。
type KindYear struct {
	Kind string
	Year int
}

// ApplyManualOverride 取得結果 auto に手入力 manual を重ねる。
// manual に1件でもある (kind, 年) は、auto の同じ組を捨てて manual で置き換える。
// 戻り値は洗い替え単位（KindYear）ごとのイベント。auto・manual のどちらにも無い組は含まれない。
func ApplyManualOverride(auto, manual []*models.MarketEvent) map[KindYear][]*models.MarketEvent {
	result := map[KindYear][]*models.MarketEvent{}
	manualKeys := map[KindYear]bool{}
	for _, e := range manual {
		k := KindYear{e.Kind, e.Date.Year()}
		manualKeys[k] = true
		result[k] = append(result[k], e)
	}
	for _, e := range auto {
		k := KindYear{e.Kind, e.Date.Year()}
		if manualKeys[k] {
			continue
		}
		result[k] = append(result[k], e)
	}
	for _, evs := range result {
		sortEvents(evs)
	}
	return result
}

// SortedKeys KindYear を kind、年の昇順に並べて返す（処理順を安定させるため）。
func SortedKeys(m map[KindYear][]*models.MarketEvent) []KindYear {
	keys := make([]KindYear, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Kind != keys[j].Kind {
			return keys[i].Kind < keys[j].Kind
		}
		return keys[i].Year < keys[j].Year
	})
	return keys
}

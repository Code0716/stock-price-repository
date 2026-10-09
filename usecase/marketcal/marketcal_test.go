package marketcal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.Local)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

func dates(events []*models.MarketEvent, kind string) []string {
	var out []string
	for _, e := range events {
		if e.Kind == kind {
			out = append(out, e.Date.Format("2006-01-02"))
		}
	}
	return out
}

func TestParseBOJ(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []string
		wantErr bool
	}{
		{
			name: "正常系: 2026/2027年の会合最終日を取り出す",
			data: readFixture(t, "boj.html"),
			want: []string{
				"2026-01-23", "2026-03-19", "2026-04-28", "2026-06-16", "2026-07-31", "2026-09-18", "2026-10-30", "2026-12-18",
				"2027-01-22", "2027-03-18", "2027-04-28", "2027-06-11", "2027-07-22", "2027-09-22", "2027-10-29", "2027-12-17",
			},
		},
		{
			name: "正常系: 1日開催の会合",
			data: []byte(`<h2 id="p2026">2026年</h2><table><tr><td>3月18日（水） [PDF]</td></tr></table>`),
			want: []string{"2026-03-18"},
		},
		{
			name: "正常系: 月またぎの2日開催",
			data: []byte(`<h2 id="p2026">2026年</h2><table><tr><td>4月30日（木）・5月 1日（金）</td></tr></table>`),
			want: []string{"2026-05-01"},
		},
		{
			name:    "異常系: 会合日が無い",
			data:    []byte(`<html><body><h2>2026年</h2><p>準備中</p></body></html>`),
			wantErr: true,
		},
		{
			name:    "異常系: 不正な日付",
			data:    []byte(`<h2>2026年</h2><table><tr><td>2月30日（月）</td></tr></table>`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBOJ(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseBOJ() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.want, dates(got, models.MarketEventKindBOJ))
			for _, e := range got {
				assert.Equal(t, models.MarketEventSourceBOJHTML, e.Source)
				assert.Equal(t, "日銀", e.Label)
			}
		})
	}
}

func TestParseFOMC(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []string
		wantErr bool
	}{
		{
			name: "正常系: 結果発表の日本時間の日付（2日目の翌日）",
			data: readFixture(t, "fomc.html"),
			want: []string{
				"2026-01-29", "2026-03-19", "2026-04-30", "2026-06-18", "2026-07-30", "2026-09-17", "2026-10-29", "2026-12-10",
				"2027-01-28", "2027-03-18", "2027-04-29", "2027-06-10", "2027-07-29", "2027-09-16", "2027-10-28", "2027-12-09",
			},
		},
		{
			name: "正常系: 月またぎの会合",
			data: []byte(`<h4><a>2026 FOMC Meetings</a></h4><div><strong>January/February</strong></div><div>31-1</div>`),
			want: []string{"2026-02-02"},
		},
		{
			name: "正常系: 年末の会合は翌年にまたがる",
			data: []byte(`<h4><a>2026 FOMC Meetings</a></h4><div><strong>December</strong></div><div>30-31</div>`),
			want: []string{"2027-01-01"},
		},
		{
			name:    "異常系: 年見出しが無い",
			data:    []byte(`<div><strong>January</strong></div><div>27-28</div>`),
			wantErr: true,
		},
		{
			name:    "異常系: 不正な日付",
			data:    []byte(`<h4><a>2026 FOMC Meetings</a></h4><div><strong>February</strong></div><div>30-31</div>`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFOMC(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFOMC() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.want, dates(got, models.MarketEventKindFOMC))
			for _, e := range got {
				assert.Equal(t, models.MarketEventSourceFedHTML, e.Source)
				assert.Equal(t, "FOMC結果", e.Label)
			}
		})
	}
}

func TestParseBLSICS(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantCPI  []string
		wantNFP  []string
		wantErr  bool
		wantOnly int
	}{
		{
			name:     "正常系: CPI・雇用統計のみ。地域別CPI・州別雇用統計は除外し、折り返し行も展開する",
			data:     readFixture(t, "bls.ics"),
			wantCPI:  []string{"2026-10-14", "2026-12-10"},
			wantNFP:  []string{"2026-11-06", "2026-12-04"},
			wantOnly: 4,
		},
		{
			name:     "正常系: CRLF 改行",
			data:     []byte("BEGIN:VEVENT\r\nDTSTART;TZID=US-Eastern:20261014T083000\r\nSUMMARY:Consumer Price Index for September 2026\r\nEND:VEVENT\r\n"),
			wantCPI:  []string{"2026-10-14"},
			wantOnly: 1,
		},
		{
			name:    "異常系: 対象イベントが無い",
			data:    []byte("BEGIN:VEVENT\nDTSTART:20261014\nSUMMARY:Producer Price Index\nEND:VEVENT\n"),
			wantErr: true,
		},
		{
			name:    "異常系: DTSTART が無い",
			data:    []byte("BEGIN:VEVENT\nSUMMARY:Employment Situation for October 2026\nEND:VEVENT\n"),
			wantErr: true,
		},
		{
			name:    "異常系: ics ではなくエラーHTML",
			data:    []byte("<html><title>Access Denied</title></html>"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBLSICS(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseBLSICS() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assert.Len(t, got, tt.wantOnly)
			assert.Equal(t, tt.wantCPI, dates(got, models.MarketEventKindUSCPI))
			assert.Equal(t, tt.wantNFP, dates(got, models.MarketEventKindUSNFP))
		})
	}
}

// calendarOf 指定期間の取引カレンダーを作る。土日と holidays は休場(0)、それ以外は営業(1)。
func calendarOf(from, to time.Time, holidays ...time.Time) []*models.MarketCalendarDay {
	hol := map[string]bool{}
	for _, h := range holidays {
		hol[dateKey(h)] = true
	}
	var days []*models.MarketCalendarDay
	for cur := from; !cur.After(to); cur = cur.AddDate(0, 0, 1) {
		div := models.MarketHolDivOpen
		if cur.Weekday() == time.Saturday || cur.Weekday() == time.Sunday || hol[dateKey(cur)] {
			div = models.MarketHolDivClosed
		}
		days = append(days, &models.MarketCalendarDay{Date: cur, HolDiv: div})
	}
	return days
}

func TestCalcSQ(t *testing.T) {
	tests := []struct {
		name      string
		days      []*models.MarketCalendarDay
		from, to  time.Time
		wantMajor []string
		wantMini  []string
	}{
		{
			name:      "正常系: 第2金曜。3/6/9/12月はメジャーSQ（2026/10/9 はミニSQ）",
			days:      calendarOf(d(2026, 1, 1), d(2026, 12, 31)),
			from:      d(2026, 9, 1),
			to:        d(2026, 12, 31),
			wantMajor: []string{"2026-09-11", "2026-12-11"},
			wantMini:  []string{"2026-10-09", "2026-11-13"},
		},
		{
			name:     "正常系: 第2金曜が休場なら直前の営業日に繰り上げる",
			days:     calendarOf(d(2026, 10, 1), d(2026, 10, 31), d(2026, 10, 9)),
			from:     d(2026, 10, 1),
			to:       d(2026, 10, 31),
			wantMini: []string{"2026-10-08"},
		},
		{
			name:     "正常系: 半日取引(2)は営業日として扱う",
			days:     []*models.MarketCalendarDay{{Date: d(2026, 10, 9), HolDiv: models.MarketHolDivHalfDay}},
			from:     d(2026, 10, 1),
			to:       d(2026, 10, 31),
			wantMini: []string{"2026-10-09"},
		},
		{
			name:     "正常系: 期間外のSQは含めない",
			days:     calendarOf(d(2026, 10, 1), d(2026, 11, 30)),
			from:     d(2026, 10, 10),
			to:       d(2026, 11, 30),
			wantMini: []string{"2026-11-13"},
		},
		{
			name: "境界: カレンダーに第2金曜が無い月はスキップ",
			days: calendarOf(d(2026, 10, 1), d(2026, 10, 8)),
			from: d(2026, 10, 1),
			to:   d(2026, 10, 31),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSQ(tt.days, tt.from, tt.to)
			assert.Equal(t, tt.wantMajor, dates(got, models.MarketEventKindSQMajor))
			assert.Equal(t, tt.wantMini, dates(got, models.MarketEventKindSQMini))
			for _, e := range got {
				assert.Equal(t, models.MarketEventSourceJQuantsCalc, e.Source)
			}
		})
	}
}

func TestParseManualEvents(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []*models.MarketEvent
		wantErr bool
	}{
		{
			name: "正常系: ラベル省略時は種別の既定ラベル",
			yaml: "- {date: \"2026-12-18\", kind: boj}\n- {date: \"2026-12-10\", kind: fomc, label: \"FOMC(手入力)\"}\n",
			want: []*models.MarketEvent{
				{Date: d(2026, 12, 10), Kind: "fomc", Label: "FOMC(手入力)", Source: "manual"},
				{Date: d(2026, 12, 18), Kind: "boj", Label: "日銀", Source: "manual"},
			},
		},
		{name: "正常系: 空ファイル", yaml: "", want: []*models.MarketEvent{}},
		{name: "異常系: 不正な kind", yaml: "- {date: \"2026-12-18\", kind: xxx}\n", wantErr: true},
		{name: "異常系: 不正な日付", yaml: "- {date: \"2026/12/18\", kind: boj}\n", wantErr: true},
		{name: "異常系: YAML 構文エラー", yaml: "- {date: \n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseManualEvents([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseManualEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func meetings(kind string, year, n int) []*models.MarketEvent {
	out := make([]*models.MarketEvent, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, newEvent(d(year, time.Month(i+1), 15), kind, "x", "src"))
	}
	return out
}

func TestValidateAnnualMeetings(t *testing.T) {
	tests := []struct {
		name     string
		events   []*models.MarketEvent
		from, to int
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "正常系: 取得できた年がすべて8件。範囲外の年は除く",
			events:  append(append(meetings("boj", 2025, 8), meetings("boj", 2026, 8)...), meetings("boj", 2027, 8)...),
			from:    2026,
			to:      2027,
			wantLen: 16,
		},
		{
			name:    "正常系: 翌年が未公表でも当年が揃っていればよい",
			events:  meetings("fomc", 2026, 8),
			from:    2026,
			to:      2027,
			wantLen: 8,
		},
		{name: "異常系: 7件の年がある", events: append(meetings("boj", 2026, 8), meetings("boj", 2027, 7)...), from: 2026, to: 2027, wantErr: true},
		{name: "異常系: 9件の年がある", events: meetings("boj", 2026, 9), from: 2026, to: 2027, wantErr: true},
		{name: "異常系: 範囲内の年が1つも無い", events: meetings("boj", 2020, 8), from: 2026, to: 2027, wantErr: true},
		{name: "異常系: 空", events: nil, from: 2026, to: 2027, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAnnualMeetings(tt.events, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAnnualMeetings() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestFilterYears(t *testing.T) {
	events := append(meetings("us_cpi", 2025, 2), append(meetings("us_cpi", 2026, 3), meetings("us_cpi", 2028, 1)...)...)
	assert.Len(t, FilterYears(events, 2026, 2027), 3)
	assert.Empty(t, FilterYears(nil, 2026, 2027))
}

func TestApplyManualOverride(t *testing.T) {
	auto := []*models.MarketEvent{
		newEvent(d(2026, 10, 30), "boj", "日銀", "boj_html"),
		newEvent(d(2026, 12, 18), "boj", "日銀", "boj_html"),
		newEvent(d(2027, 1, 22), "boj", "日銀", "boj_html"),
		newEvent(d(2026, 10, 14), "us_cpi", "米CPI", "bls_ics"),
	}
	manual := []*models.MarketEvent{
		newEvent(d(2026, 12, 17), "boj", "日銀", "manual"),
		newEvent(d(2026, 7, 10), "fomc", "FOMC結果", "manual"),
	}

	got := ApplyManualOverride(auto, manual)

	// 手入力のある (boj, 2026) は自動取得分を捨てて置き換える。2027 は自動取得のまま。
	assert.Equal(t, []*models.MarketEvent{manual[0]}, got[KindYear{"boj", 2026}])
	assert.Equal(t, []*models.MarketEvent{auto[2]}, got[KindYear{"boj", 2027}])
	// 手入力だけの組（自動取得に無い）も洗い替え単位になる。
	assert.Equal(t, []*models.MarketEvent{manual[1]}, got[KindYear{"fomc", 2026}])
	assert.Equal(t, []*models.MarketEvent{auto[3]}, got[KindYear{"us_cpi", 2026}])
	assert.Len(t, got, 4)

	assert.Equal(t,
		[]KindYear{{"boj", 2026}, {"boj", 2027}, {"fomc", 2026}, {"us_cpi", 2026}},
		SortedKeys(got),
	)
}

func TestApplyManualOverride_Empty(t *testing.T) {
	assert.Empty(t, ApplyManualOverride(nil, nil))
}

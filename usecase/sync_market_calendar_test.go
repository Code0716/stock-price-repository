package usecase

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func marketcalFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("marketcal", "testdata", name))
	require.NoError(t, err)
	return b
}

// tradingCalendarOf 土日を休場(0)、それ以外を営業(1)とした取引カレンダー。
func tradingCalendarOf(from, to time.Time) []*gateway.TradingCalendarDay {
	var days []*gateway.TradingCalendarDay
	for cur := from; !cur.After(to); cur = cur.AddDate(0, 0, 1) {
		div := models.MarketHolDivOpen
		if cur.Weekday() == time.Saturday || cur.Weekday() == time.Sunday {
			div = models.MarketHolDivClosed
		}
		days = append(days, &gateway.TradingCalendarDay{Date: cur, HolDiv: div})
	}
	return days
}

// replaceCall ReplaceEvents の呼び出し記録。
type replaceCall struct {
	kind  string
	year  int
	dates []string
	src   string
}

func TestSyncMarketCalendarInteractor_SyncMarketCalendar(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.Local)
	calFrom := now.AddDate(-1, 0, 0)
	calTo := time.Date(2027, 12, 31, 0, 0, 0, 0, time.Local)

	bojHTML := marketcalFixture(t, "boj.html")
	fomcHTML := marketcalFixture(t, "fomc.html")
	blsICS := marketcalFixture(t, "bls.ics")

	type fetchResult struct {
		body []byte
		err  error
	}
	type scenario struct {
		calendarErr error
		boj         fetchResult
		fomc        fetchResult
		bls         fetchResult
		manual      string
		replaceErr  error
	}
	ok := func(b []byte) fetchResult { return fetchResult{body: b} }

	tests := []struct {
		name         string
		sc           scenario
		wantErrParts []string
		wantUpsert   bool
		wantKinds    []string // "kind/year" のうち洗い替えされるもの
		check        func(t *testing.T, calls map[string]replaceCall)
	}{
		{
			name:       "正常系: 全ソース成功で (種別, 年) ごとに洗い替える",
			sc:         scenario{boj: ok(bojHTML), fomc: ok(fomcHTML), bls: ok(blsICS)},
			wantUpsert: true,
			wantKinds: []string{
				"boj/2026", "boj/2027", "fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026",
				"sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027",
			},
			check: func(t *testing.T, calls map[string]replaceCall) {
				assert.Len(t, calls["boj/2026"].dates, 8)
				assert.Equal(t, "2026-10-30", calls["boj/2026"].dates[6])
				assert.Equal(t, "boj_html", calls["boj/2026"].src)
				assert.Len(t, calls["fomc/2027"].dates, 8)
				assert.Equal(t, []string{"2026-10-14", "2026-11-10", "2026-12-10"}, calls["us_cpi/2026"].dates)
				assert.Equal(t, []string{"2026-10-02", "2026-11-06", "2026-12-04"}, calls["us_nfp/2026"].dates)
				// 2026/10/9 はミニSQ、12月はメジャーSQ（第2金曜 12/11）
				assert.Contains(t, calls["sq_mini/2026"].dates, "2026-10-09")
				assert.Equal(t, []string{"2026-03-13", "2026-06-12", "2026-09-11", "2026-12-11"}, calls["sq_major/2026"].dates)
				assert.Equal(t, "jquants_calc", calls["sq_mini/2026"].src)
			},
		},
		{
			name:         "異常系: BLS が 403 でも他は更新し、BLS の既存データは触らない",
			sc:           scenario{boj: ok(bojHTML), fomc: ok(fomcHTML), bls: fetchResult{err: errors.New("status 403")}},
			wantUpsert:   true,
			wantKinds:    []string{"boj/2026", "boj/2027", "fomc/2026", "fomc/2027", "sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027"},
			wantErrParts: []string{"bls_ics", "403"},
		},
		{
			name:         "異常系: 日銀ページの構造が壊れたら日銀だけ更新しない",
			sc:           scenario{boj: ok([]byte("<html><body>メンテナンス中</body></html>")), fomc: ok(fomcHTML), bls: ok(blsICS)},
			wantUpsert:   true,
			wantKinds:    []string{"fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026", "sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027"},
			wantErrParts: []string{"boj_html"},
		},
		{
			name: "異常系: 日銀の会合数が年8件でなければ日銀だけ更新しない",
			sc: scenario{
				boj:  ok([]byte(`<h2 id="p2026">2026年</h2><table><tr><td>1月22日（木）・23日（金）</td></tr></table>`)),
				fomc: ok(fomcHTML), bls: ok(blsICS),
			},
			wantUpsert:   true,
			wantKinds:    []string{"fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026", "sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027"},
			wantErrParts: []string{"boj_html", "8"},
		},
		{
			name:         "異常系: J-Quants カレンダー取得失敗ならカレンダー保存もSQ更新もしない",
			sc:           scenario{calendarErr: errors.New("status 500"), boj: ok(bojHTML), fomc: ok(fomcHTML), bls: ok(blsICS)},
			wantKinds:    []string{"boj/2026", "boj/2027", "fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026"},
			wantErrParts: []string{"取引カレンダー", "500"},
		},
		{
			name: "異常系: 全ソース失敗でも手入力は反映する",
			sc: scenario{
				calendarErr: errors.New("x"), boj: fetchResult{err: errors.New("x")}, fomc: fetchResult{err: errors.New("x")}, bls: fetchResult{err: errors.New("x")},
				manual: "- {date: \"2026-12-17\", kind: boj}\n",
			},
			wantKinds:    []string{"boj/2026"},
			wantErrParts: []string{"取引カレンダー", "boj_html", "fed_html", "bls_ics"},
			check: func(t *testing.T, calls map[string]replaceCall) {
				assert.Equal(t, []string{"2026-12-17"}, calls["boj/2026"].dates)
				assert.Equal(t, "manual", calls["boj/2026"].src)
			},
		},
		{
			name: "正常系: 手入力がある (種別, 年) は取得結果を捨てて置き換える",
			sc: scenario{
				boj: ok(bojHTML), fomc: ok(fomcHTML), bls: ok(blsICS),
				manual: "- {date: \"2026-12-17\", kind: boj}\n- {date: \"2026-10-29\", kind: boj, label: \"日銀(臨時)\"}\n",
			},
			wantUpsert: true,
			wantKinds: []string{
				"boj/2026", "boj/2027", "fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026",
				"sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027",
			},
			check: func(t *testing.T, calls map[string]replaceCall) {
				assert.Equal(t, []string{"2026-10-29", "2026-12-17"}, calls["boj/2026"].dates)
				assert.Equal(t, "manual", calls["boj/2026"].src)
				assert.Len(t, calls["boj/2027"].dates, 8, "手入力の無い年は取得結果のまま")
				assert.Equal(t, "boj_html", calls["boj/2027"].src)
			},
		},
		{
			name:         "異常系: 手入力YAMLが不正でも自動取得分は更新する",
			sc:           scenario{boj: ok(bojHTML), fomc: ok(fomcHTML), bls: ok(blsICS), manual: "- {date: \"2026-12-17\", kind: xxx}\n"},
			wantUpsert:   true,
			wantKinds:    []string{"boj/2026", "boj/2027", "fomc/2026", "fomc/2027", "us_cpi/2026", "us_nfp/2026", "sq_major/2026", "sq_major/2027", "sq_mini/2026", "sq_mini/2027"},
			wantErrParts: []string{"manual", "kind"},
		},
		{
			name:         "異常系: 保存に失敗したらエラーを返す",
			sc:           scenario{boj: ok(bojHTML), fomc: ok(fomcHTML), bls: ok(blsICS), replaceErr: errors.New("db down")},
			wantUpsert:   true,
			wantKinds:    nil,
			wantErrParts: []string{"市場イベント保存", "db down"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stockAPI := mock_gateway.NewMockStockAPIClient(ctrl)
			if tt.sc.calendarErr != nil {
				stockAPI.EXPECT().GetTradingCalendar(gomock.Any(), gomock.Eq(calFrom), gomock.Eq(calTo)).Return(nil, tt.sc.calendarErr)
			} else {
				stockAPI.EXPECT().GetTradingCalendar(gomock.Any(), gomock.Eq(calFrom), gomock.Eq(calTo)).Return(tradingCalendarOf(calFrom, calTo), nil)
			}

			eco := mock_gateway.NewMockEconomicCalendarClient(ctrl)
			eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return(tt.sc.boj.body, tt.sc.boj.err)
			eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(tt.sc.fomc.body, tt.sc.fomc.err)
			eco.EXPECT().FetchBLSICS(gomock.Any()).Return(tt.sc.bls.body, tt.sc.bls.err)

			repo := mock_repositories.NewMockMarketCalendarRepository(ctrl)
			if tt.wantUpsert {
				repo.EXPECT().UpsertDays(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, days []*models.MarketCalendarDay) error {
					assert.Len(t, days, int(calTo.Sub(calFrom).Hours()/24)+1)
					return nil
				})
			}
			calls := map[string]replaceCall{}
			repo.EXPECT().ReplaceEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, kind string, from, to time.Time, events []*models.MarketEvent) error {
					assert.Equal(t, time.January, from.Month(), "年初から")
					assert.Equal(t, 1, from.Day(), "年初から")
					assert.Equal(t, from.Year(), to.Year())
					assert.Equal(t, time.December, to.Month())
					c := replaceCall{kind: kind, year: from.Year()}
					for _, e := range events {
						assert.Equal(t, kind, e.Kind)
						assert.Equal(t, from.Year(), e.Date.Year())
						c.dates = append(c.dates, e.Date.Format("2006-01-02"))
						c.src = e.Source
					}
					calls[kind+"/"+from.Format("2006")] = c
					return tt.sc.replaceErr
				}).AnyTimes()

			tx := mock_repositories.NewMockTransaction(ctrl)
			tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).AnyTimes()

			si := NewSyncMarketCalendarInteractor(tx, stockAPI, eco, repo)
			err := si.SyncMarketCalendar(context.Background(), now, []byte(tt.sc.manual))

			if len(tt.wantErrParts) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, p := range tt.wantErrParts {
					assert.Contains(t, err.Error(), p)
				}
			}

			if tt.sc.replaceErr == nil {
				got := make([]string, 0, len(calls))
				for k := range calls {
					got = append(got, k)
				}
				want := append([]string(nil), tt.wantKinds...)
				sort.Strings(got)
				sort.Strings(want)
				assert.Equal(t, want, got)
			}
			if tt.check != nil {
				tt.check(t, calls)
			}
		})
	}
}

func TestSyncMarketCalendarInteractor_SyncMarketCalendar_EmptyCalendar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.Local)
	stockAPI := mock_gateway.NewMockStockAPIClient(ctrl)
	stockAPI.EXPECT().GetTradingCalendar(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.TradingCalendarDay{}, nil)
	eco := mock_gateway.NewMockEconomicCalendarClient(ctrl)
	eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return(nil, errors.New("x"))
	eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(nil, errors.New("x"))
	eco.EXPECT().FetchBLSICS(gomock.Any()).Return(nil, errors.New("x"))
	repo := mock_repositories.NewMockMarketCalendarRepository(ctrl) // UpsertDays / ReplaceEvents は呼ばれない
	tx := mock_repositories.NewMockTransaction(ctrl)

	err := NewSyncMarketCalendarInteractor(tx, stockAPI, eco, repo).SyncMarketCalendar(context.Background(), now, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "0件")
}

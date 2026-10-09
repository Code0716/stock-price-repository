package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	"github.com/Code0716/stock-price-repository/entrypoint/api/router"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func marketCalFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "usecase", "marketcal", "testdata", name))
	require.NoError(t, err)
	return b
}

// marketCalTradingDays 土日を休場(0)、2026/10/12(スポーツの日)を休場(0)、それ以外を営業(1)とした取引カレンダー。
func marketCalTradingDays(from, to time.Time) []*gateway.TradingCalendarDay {
	holiday := time.Date(2026, 10, 12, 0, 0, 0, 0, time.Local)
	var days []*gateway.TradingCalendarDay
	// 実際の gateway は日付文字列をパースして 0 時の日付を返すので、from の時刻は切り捨てる。
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
	for cur := start; !cur.After(to); cur = cur.AddDate(0, 0, 1) {
		div := models.MarketHolDivOpen
		if cur.Weekday() == time.Saturday || cur.Weekday() == time.Sunday || cur.Equal(holiday) {
			div = models.MarketHolDivClosed
		}
		days = append(days, &gateway.TradingCalendarDay{Date: cur, HolDiv: div})
	}
	return days
}

type marketCalendarAPIResponse struct {
	Days []struct {
		Date   string `json:"date"`
		HolDiv int    `json:"holDiv"`
	} `json:"days"`
	Events []struct {
		Date  string `json:"date"`
		Kind  string `json:"kind"`
		Label string `json:"label"`
	} `json:"events"`
}

func getMarketCalendar(t *testing.T, baseURL, query string) (int, marketCalendarAPIResponse) {
	t.Helper()
	resp, err := http.Get(baseURL + "/market/calendar?" + query)
	require.NoError(t, err)
	defer resp.Body.Close()
	var out marketCalendarAPIResponse
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	}
	return resp.StatusCode, out
}

func TestE2E_MarketCalendar(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.Local)
	calFrom := now.AddDate(-1, 0, 0)
	calTo := time.Date(2027, 12, 31, 0, 0, 0, 0, time.Local)

	repo := database.NewMarketCalendarRepositoryImpl(db)
	tx := database.NewTransaction(db)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stockAPI := mock_gateway.NewMockStockAPIClient(ctrl)
	stockAPI.EXPECT().GetTradingCalendar(gomock.Any(), gomock.Eq(calFrom), gomock.Eq(calTo)).
		Return(marketCalTradingDays(calFrom, calTo), nil).AnyTimes()
	eco := mock_gateway.NewMockEconomicCalendarClient(ctrl)

	syncInteractor := usecase.NewSyncMarketCalendarInteractor(tx, stockAPI, eco, repo)
	mux := router.NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		handler.NewMarketCalendarHandler(usecase.NewMarketCalendarInteractor(repo), zap.NewNop()))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	eventKey := func(r marketCalendarAPIResponse) []string {
		var out []string
		for _, e := range r.Events {
			out = append(out, e.Date+" "+e.Kind+" "+e.Label)
		}
		return out
	}

	t.Run("同期した営業日とイベントをAPIで取得できる", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return(marketCalFixture(t, "boj.html"), nil)
		eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(marketCalFixture(t, "fomc.html"), nil)
		eco.EXPECT().FetchBLSICS(gomock.Any()).Return(marketCalFixture(t, "bls.ics"), nil)

		require.NoError(t, syncInteractor.SyncMarketCalendar(ctx, now, nil))

		status, got := getMarketCalendar(t, ts.URL, "from=2026-10-01&to=2026-10-31")
		require.Equal(t, http.StatusOK, status)

		// 営業日・休場日
		holDiv := map[string]int{}
		for _, d := range got.Days {
			holDiv[d.Date] = d.HolDiv
		}
		assert.Len(t, got.Days, 31)
		assert.Equal(t, 1, holDiv["2026-10-09"])
		assert.Equal(t, 0, holDiv["2026-10-10"], "土曜は休場")
		assert.Equal(t, 0, holDiv["2026-10-12"], "祝日は休場")

		// イベント（日付昇順）
		assert.Equal(t, []string{
			"2026-10-09 sq_mini SQ",
			"2026-10-14 us_cpi 米CPI",
			"2026-10-29 fomc FOMC結果",
			"2026-10-30 boj 日銀",
		}, eventKey(got))
	})

	t.Run("再実行しても行が増えない（冪等）", func(t *testing.T) {
		eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return(marketCalFixture(t, "boj.html"), nil)
		eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(marketCalFixture(t, "fomc.html"), nil)
		eco.EXPECT().FetchBLSICS(gomock.Any()).Return(marketCalFixture(t, "bls.ics"), nil)
		before, err := repo.ListEvents(ctx, calFrom, calTo)
		require.NoError(t, err)

		require.NoError(t, syncInteractor.SyncMarketCalendar(ctx, now, nil))

		after, err := repo.ListEvents(ctx, calFrom, calTo)
		require.NoError(t, err)
		assert.Len(t, after, len(before))
	})

	t.Run("一部のソースが壊れてもエラーを返しつつ、そのソースの既存データは残る", func(t *testing.T) {
		// 直前のサブテストで全ソースが入った状態から、BLS だけ 403、日銀はページ構造が壊れた状態で再同期する。
		eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return([]byte("<html>メンテナンス中</html>"), nil)
		eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(marketCalFixture(t, "fomc.html"), nil)
		eco.EXPECT().FetchBLSICS(gomock.Any()).Return(nil, errors.New("status 403"))

		err := syncInteractor.SyncMarketCalendar(ctx, now, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boj_html")
		assert.Contains(t, err.Error(), "bls_ics")

		_, got := getMarketCalendar(t, ts.URL, "from=2026-10-01&to=2026-10-31")
		assert.Equal(t, []string{
			"2026-10-09 sq_mini SQ",
			"2026-10-14 us_cpi 米CPI",
			"2026-10-29 fomc FOMC結果",
			"2026-10-30 boj 日銀",
		}, eventKey(got), "壊れたソースの既存データは消えない")
	})

	t.Run("手入力YAMLで日銀の年を丸ごと置き換えられる", func(t *testing.T) {
		eco.EXPECT().FetchBOJSchedulePage(gomock.Any()).Return([]byte("<html>メンテナンス中</html>"), nil)
		eco.EXPECT().FetchFOMCPage(gomock.Any()).Return(marketCalFixture(t, "fomc.html"), nil)
		eco.EXPECT().FetchBLSICS(gomock.Any()).Return(marketCalFixture(t, "bls.ics"), nil)

		manual := []byte("- {date: \"2026-10-31\", kind: boj}\n")
		err := syncInteractor.SyncMarketCalendar(ctx, now, manual)
		require.Error(t, err, "日銀の取得失敗は引き続きエラーとして通知する")

		_, got := getMarketCalendar(t, ts.URL, "from=2026-10-01&to=2026-12-31")
		var boj []string
		for _, e := range got.Events {
			if e.Kind == "boj" {
				boj = append(boj, e.Date)
			}
		}
		assert.Equal(t, []string{"2026-10-31"}, boj, "2026年の日銀は手入力だけになる")
	})

	t.Run("APIの入力検証", func(t *testing.T) {
		status, _ := getMarketCalendar(t, ts.URL, "from=2026-10-01")
		assert.Equal(t, http.StatusBadRequest, status)

		status, got := getMarketCalendar(t, ts.URL, "from=2030-01-01&to=2030-01-31")
		assert.Equal(t, http.StatusOK, status)
		assert.Empty(t, got.Days)
		assert.Empty(t, got.Events)
	})
}

// TestE2E_SyncMarketCalendarCommand runner に sync_market_calendar が登録され、usecase が呼ばれることを確認する。
func TestE2E_SyncMarketCalendarCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	syncInteractor := mock_usecase.NewMockSyncMarketCalendarInteractor(ctrl)
	syncInteractor.EXPECT().SyncMarketCalendar(gomock.Any(), gomock.Any(), gomock.Nil()).Return(nil)

	mockSlackAPI := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	mockSlackAPI.EXPECT().
		SendBlockMessage(gomock.Any(), gateway.SlackChannelNameDevNotification, gomock.Any()).
		Return(nil).
		AnyTimes()

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		SyncMarketCalendarCommand: commands.NewSyncMarketCalendarCommand(syncInteractor),
		SlackAPIClient:            mockSlackAPI,
	})

	err := runner.Run(context.Background(), []string{"main", "sync_market_calendar", "--manual", filepath.Join(t.TempDir(), "none.yaml")})
	assert.NoError(t, err)
}

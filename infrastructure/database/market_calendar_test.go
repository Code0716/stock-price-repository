package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
)

func mcDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func TestMarketCalendarRepositoryImpl_Days(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewMarketCalendarRepositoryImpl(db)
	ctx := context.Background()

	t.Run("空の upsert は何もしない", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		require.NoError(t, repo.UpsertDays(ctx, nil))
		got, err := repo.ListDays(ctx, mcDate(2026, 1, 1), mcDate(2026, 12, 31))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("upsert は日付キーで上書きし、範囲（両端含む）で昇順に取得する", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		require.NoError(t, repo.UpsertDays(ctx, []*models.MarketCalendarDay{
			{Date: mcDate(2026, 10, 12), HolDiv: models.MarketHolDivClosed},
			{Date: mcDate(2026, 10, 9), HolDiv: models.MarketHolDivOpen},
			{Date: mcDate(2026, 10, 10), HolDiv: models.MarketHolDivClosed},
			{Date: mcDate(2026, 11, 3), HolDiv: models.MarketHolDivOpen},
		}))
		// 同じ日付を別の区分で upsert → 上書きされる
		require.NoError(t, repo.UpsertDays(ctx, []*models.MarketCalendarDay{
			{Date: mcDate(2026, 10, 12), HolDiv: models.MarketHolDivHolidayTrades},
		}))

		got, err := repo.ListDays(ctx, mcDate(2026, 10, 9), mcDate(2026, 10, 12))
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "2026-10-09", got[0].Date.Format("2006-01-02"))
		assert.Equal(t, models.MarketHolDivOpen, got[0].HolDiv)
		assert.Equal(t, "2026-10-10", got[1].Date.Format("2006-01-02"))
		assert.Equal(t, "2026-10-12", got[2].Date.Format("2006-01-02"))
		assert.Equal(t, models.MarketHolDivHolidayTrades, got[2].HolDiv)
	})
}

func TestMarketCalendarRepositoryImpl_Events(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewMarketCalendarRepositoryImpl(db)
	ctx := context.Background()

	ev := func(date time.Time, kind, label, src string) *models.MarketEvent {
		return &models.MarketEvent{Date: date, Kind: kind, Label: label, Source: src}
	}
	y := func(year int) (time.Time, time.Time) { return mcDate(year, 1, 1), mcDate(year, 12, 31) }
	from26, to26 := y(2026)
	from27, to27 := y(2027)

	t.Run("洗い替えは同じ種別・同じ年だけを削除して作り直す", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindBOJ, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 30), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceBOJHTML),
			ev(mcDate(2026, 12, 18), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceBOJHTML),
		}))
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindBOJ, from27, to27, []*models.MarketEvent{
			ev(mcDate(2027, 1, 22), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceBOJHTML),
		}))
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindUSCPI, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 14), models.MarketEventKindUSCPI, "米CPI", models.MarketEventSourceBLSICS),
		}))

		// 2026年の日銀だけ手入力で置き換える
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindBOJ, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 12, 17), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceManual),
		}))

		got, err := repo.ListEvents(ctx, mcDate(2026, 1, 1), mcDate(2027, 12, 31))
		require.NoError(t, err)
		var summary []string
		for _, e := range got {
			summary = append(summary, e.Date.Format("2006-01-02")+" "+e.Kind+" "+e.Source)
		}
		assert.Equal(t, []string{
			"2026-10-14 us_cpi bls_ics",
			"2026-12-17 boj manual",
			"2027-01-22 boj boj_html",
		}, summary)
	})

	t.Run("events が空なら削除だけ行う", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindFOMC, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 29), models.MarketEventKindFOMC, "FOMC結果", models.MarketEventSourceFedHTML),
		}))
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindFOMC, from26, to26, nil))

		got, err := repo.ListEvents(ctx, from26, to26)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("同じ日に複数種別があっても取得でき、範囲外は除く", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindSQMini, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 9), models.MarketEventKindSQMini, "SQ", models.MarketEventSourceJQuantsCalc),
		}))
		require.NoError(t, repo.ReplaceEvents(ctx, models.MarketEventKindUSNFP, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 9), models.MarketEventKindUSNFP, "米雇用統計", models.MarketEventSourceBLSICS),
			ev(mcDate(2026, 11, 6), models.MarketEventKindUSNFP, "米雇用統計", models.MarketEventSourceBLSICS),
		}))

		got, err := repo.ListEvents(ctx, mcDate(2026, 10, 1), mcDate(2026, 10, 31))
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, models.MarketEventKindSQMini, got[0].Kind, "同日は kind 昇順")
		assert.Equal(t, models.MarketEventKindUSNFP, got[1].Kind)
	})

	t.Run("同じ (日付, 種別) を重複して作るとエラー", func(t *testing.T) {
		helper.TruncateAllTables(t, db)
		err := repo.ReplaceEvents(ctx, models.MarketEventKindBOJ, from26, to26, []*models.MarketEvent{
			ev(mcDate(2026, 10, 30), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceBOJHTML),
			ev(mcDate(2026, 10, 30), models.MarketEventKindBOJ, "日銀", models.MarketEventSourceBOJHTML),
		})
		assert.Error(t, err)
	})
}

package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/domain_service"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func viewTestPick(pickDate time.Time, rank int, brandID, symbol, score string, strategies []string) *models.DailyStockPick {
	return &models.DailyStockPick{
		PickDate:     pickDate,
		StockBrandID: brandID,
		TickerSymbol: symbol,
		PickRank:     rank,
		Score:        decimal.RequireFromString(score),
		ScoreVersion: domain_service.DailyPickScoreVersion,
		SignalCount:  len(strategies),
		Strategies:   strategies,
	}
}

func TestDailyStockPickInteractorImpl_GetDay(t *testing.T) {
	pickDate := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("dateがnilなら最新pick_dateを引く", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().FindLatestPickDate(gomock.Any()).Return(&pickDate, nil)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return([]*models.DailyStockPick{
			viewTestPick(pickDate, 1, "b1", "1000", "82.5", []string{"macd_bullish"}),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Eq([]string{"b1"})).Return([]*models.StockBrand{
			{ID: "b1", Name: "テスト銘柄"},
		}, nil)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), nil)
		assert.NoError(t, err)
		assert.NotNil(t, got.PickDate)
		assert.Equal(t, "2026-07-24", *got.PickDate)
		assert.Len(t, got.Items, 1)
		assert.Equal(t, "テスト銘柄", got.Items[0].Name)
	})

	t.Run("推奨が1件も無ければListByPickDateを呼ばずPickDate=nilで返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().FindLatestPickDate(gomock.Any()).Return(nil, nil)
		// ListByPickDate は呼ばれない

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), nil)
		assert.NoError(t, err)
		assert.Nil(t, got.PickDate)
		assert.NotNil(t, got.Items, "nilではなく空スライスを返す（JSONがnullにならないように）")
		assert.Empty(t, got.Items)
	})

	t.Run("指定日にデータが無ければ空で返す（休場日）", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return(nil, nil)
		// date が指定されているので FindLatestPickDate は呼ばれない

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), &pickDate)
		assert.NoError(t, err)
		assert.Nil(t, got.PickDate)
		assert.Empty(t, got.Items)
	})

	t.Run("FindByIDsに無いIDはNameが空のまま", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return([]*models.DailyStockPick{
			viewTestPick(pickDate, 1, "b1", "1000", "82.5", []string{"macd_bullish"}),
			viewTestPick(pickDate, 2, "b2", "2000", "70.0", []string{"ma_cross"}),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Eq([]string{"b1", "b2"})).Return([]*models.StockBrand{
			{ID: "b1", Name: "テスト銘柄"},
		}, nil)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), &pickDate)
		assert.NoError(t, err)
		assert.Equal(t, "テスト銘柄", got.Items[0].Name)
		assert.Equal(t, "", got.Items[1].Name)
	})

	t.Run("戦略キーに日本語ラベルが付き、未知キーはキーがそのままラベルになる", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return([]*models.DailyStockPick{
			viewTestPick(pickDate, 1, "b1", "1000", "82.5", []string{"macd_bullish", "unknown_strategy"}),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return([]*models.StockBrand{{ID: "b1", Name: "テスト銘柄"}}, nil)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), &pickDate)
		assert.NoError(t, err)
		assert.Equal(t, "macd_bullish", got.Items[0].Strategies[0].Key)
		assert.Equal(t, "MACD強気", got.Items[0].Strategies[0].Label)
		assert.Equal(t, "unknown_strategy", got.Items[0].Strategies[1].Label, "未知キーはキーをそのままラベルにする")
	})

	t.Run("全件答え合わせ済みならEvaluated=true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		evaluatedAt := pickDate
		outcome := models.DailyStockPickOutcomeWin
		p := viewTestPick(pickDate, 1, "b1", "1000", "82.5", []string{"macd_bullish"})
		p.EvaluatedAt = &evaluatedAt
		p.Outcome = &outcome

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return([]*models.DailyStockPick{p}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return([]*models.StockBrand{{ID: "b1", Name: "テスト銘柄"}}, nil)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), &pickDate)
		assert.NoError(t, err)
		assert.True(t, got.Evaluated)
		assert.NotNil(t, got.Items[0].EvaluatedAt)
	})

	t.Run("未確定が混ざればEvaluated=false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByPickDate(gomock.Any(), gomock.Eq(pickDate)).Return([]*models.DailyStockPick{
			viewTestPick(pickDate, 1, "b1", "1000", "82.5", []string{"macd_bullish"}),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return([]*models.StockBrand{{ID: "b1", Name: "テスト銘柄"}}, nil)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetDay(context.Background(), &pickDate)
		assert.NoError(t, err)
		assert.False(t, got.Evaluated)
		assert.Equal(t, 1, got.Summary.PendingCount)
		assert.Nil(t, got.Items[0].EvaluatedAt)
	})
}

func TestDailyStockPickInteractorImpl_GetPickDates(t *testing.T) {
	t.Run("新しい順の日付文字列を返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPickDates(gomock.Any(), gomock.Eq(30)).Return([]time.Time{
			time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetPickDates(context.Background(), 30)
		assert.NoError(t, err)
		assert.Equal(t, []string{"2026-07-24", "2026-07-23"}, got.Dates)
	})

	t.Run("limit<=0なら既定値を使う", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPickDates(gomock.Any(), gomock.Eq(dailyStockPickDatesDefaultLimit)).Return(nil, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetPickDates(context.Background(), 0)
		assert.NoError(t, err)
		assert.NotNil(t, got.Dates, "nilではなく空スライス")
		assert.Empty(t, got.Dates)
	})
}

func TestDailyStockPickInteractorImpl_GetStats(t *testing.T) {
	d1 := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("scoreVersionが空なら現行バージョンで絞る", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().
			ListByDateRange(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Eq(domain_service.DailyPickScoreVersion)).
			Return(nil, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetStats(context.Background(), nil, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, domain_service.DailyPickScoreVersion, got.ScoreVersion)
		assert.NotNil(t, got.Daily)
		assert.Empty(t, got.Daily)
		assert.NotNil(t, got.ScoreBands)
		assert.Empty(t, got.ScoreBands)
		assert.Nil(t, got.From, "データ0件ならfrom/toはnull")
		assert.Nil(t, got.To)
	})

	t.Run("指定されたscoreVersionをそのまま使う", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().
			ListByDateRange(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Eq("v2")).
			Return(nil, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetStats(context.Background(), nil, nil, "v2")
		assert.NoError(t, err)
		assert.Equal(t, "v2", got.ScoreVersion)
	})

	t.Run("from/toは実データの範囲を返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListByDateRange(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Any()).Return([]*models.DailyStockPick{
			viewTestPick(d1, 1, "b1", "1000", "82.5", []string{"macd_bullish"}),
			viewTestPick(d2, 1, "b2", "2000", "65.0", []string{"ma_cross"}),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyStockPickInteractor(pickRepo, brandRepo).GetStats(context.Background(), nil, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, "2026-07-23", *got.From)
		assert.Equal(t, "2026-07-24", *got.To)
		assert.Len(t, got.Daily, 2)
		assert.Len(t, got.ScoreBands, 2, "60帯と80帯")
		assert.Equal(t, 2, got.Totals.Total)
	})
}

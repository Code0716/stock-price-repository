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

func avoidTestRow(asOfDate time.Time, rank int, brandID, symbol string, severity models.DailyAvoidSeverity, vol string) *models.DailyAvoidStock {
	return &models.DailyAvoidStock{
		AsOfDate:             asOfDate,
		StockBrandID:         brandID,
		TickerSymbol:         symbol,
		AvoidRank:            rank,
		Severity:             severity,
		Reason:               models.DailyAvoidReasonHighVolatility,
		RuleVersion:          domain_service.DailyAvoidRuleVersion,
		Volatility12M:        decimal.RequireFromString(vol),
		VolatilityPercentile: decimal.NewFromInt(int64(rank)).Div(decimal.NewFromInt(100)),
		UniverseSize:         100,
		ThresholdVolatility:  decimal.RequireFromString("0.35"),
		AvgTradingValue:      decimal.NewFromInt(200_000_000),
		BaseClosePrice:       decimal.NewFromInt(1500),
	}
}

func TestDailyAvoidStockInteractorImpl_GetDay(t *testing.T) {
	asOfDate := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("dateがnilなら最新as_of_dateを引く", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().FindLatestAsOfDate(gomock.Any()).Return(&asOfDate, nil)
		avoidRepo.EXPECT().ListByAsOfDate(gomock.Any(), gomock.Eq(asOfDate)).Return([]*models.DailyAvoidStock{
			avoidTestRow(asOfDate, 1, "b1", "1000", models.DailyAvoidSeverityHigh, "0.55"),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Eq([]string{"b1"})).Return([]*models.StockBrand{
			{ID: "b1", Name: "テスト銘柄"},
		}, nil)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetDay(context.Background(), nil)
		assert.NoError(t, err)
		assert.NotNil(t, got.AsOfDate)
		assert.Equal(t, "2026-07-24", *got.AsOfDate)
		assert.Len(t, got.Items, 1)
		assert.Equal(t, "テスト銘柄", got.Items[0].Name)
	})

	t.Run("該当が1件も無ければListByAsOfDateを呼ばずAsOfDate=nilで返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().FindLatestAsOfDate(gomock.Any()).Return(nil, nil)
		// ListByAsOfDate は呼ばれない

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetDay(context.Background(), nil)
		assert.NoError(t, err)
		assert.Nil(t, got.AsOfDate)
		assert.NotNil(t, got.Items, "nilではなく空スライスを返す（JSONがnullにならないように）")
		assert.Empty(t, got.Items)
		assert.Equal(t, domain_service.DailyAvoidRuleVersion, got.RuleVersion)
	})

	t.Run("指定日にデータが無ければ空で返す（休場日）", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ListByAsOfDate(gomock.Any(), gomock.Eq(asOfDate)).Return(nil, nil)
		// date が指定されているので FindLatestAsOfDate は呼ばれない

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetDay(context.Background(), &asOfDate)
		assert.NoError(t, err)
		assert.Nil(t, got.AsOfDate)
		assert.Empty(t, got.Items)
	})

	t.Run("FindByIDsに無いIDはNameが空のまま", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ListByAsOfDate(gomock.Any(), gomock.Eq(asOfDate)).Return([]*models.DailyAvoidStock{
			avoidTestRow(asOfDate, 1, "b1", "1000", models.DailyAvoidSeverityHigh, "0.55"),
			avoidTestRow(asOfDate, 2, "b2", "2000", models.DailyAvoidSeverityElevated, "0.45"),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Eq([]string{"b1", "b2"})).Return([]*models.StockBrand{
			{ID: "b1", Name: "テスト銘柄"},
		}, nil)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetDay(context.Background(), &asOfDate)
		assert.NoError(t, err)
		assert.Equal(t, "テスト銘柄", got.Items[0].Name)
		assert.Equal(t, "", got.Items[1].Name)
	})

	t.Run("summaryとuniverseSize/thresholdVolatilityが先頭行から詰められる", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rows := []*models.DailyAvoidStock{
			avoidTestRow(asOfDate, 1, "b1", "1000", models.DailyAvoidSeverityHigh, "0.55"),
			avoidTestRow(asOfDate, 2, "b2", "2000", models.DailyAvoidSeverityElevated, "0.45"),
		}
		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ListByAsOfDate(gomock.Any(), gomock.Eq(asOfDate)).Return(rows, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return([]*models.StockBrand{
			{ID: "b1", Name: "テスト銘柄1"},
			{ID: "b2", Name: "テスト銘柄2"},
		}, nil)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetDay(context.Background(), &asOfDate)
		assert.NoError(t, err)
		assert.Equal(t, 100, got.UniverseSize)
		assert.True(t, got.ThresholdVolatility.Equal(decimal.RequireFromString("0.35")))
		assert.Equal(t, 2, got.Summary.FlaggedCount)
		assert.Equal(t, 1, got.Summary.HighCount)
		assert.Equal(t, 1, got.Summary.ElevatedCount)
	})
}

func TestDailyAvoidStockInteractorImpl_GetAsOfDates(t *testing.T) {
	t.Run("新しい順の日付文字列を返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ListAsOfDates(gomock.Any(), gomock.Eq(30)).Return([]time.Time{
			time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		}, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetAsOfDates(context.Background(), 30)
		assert.NoError(t, err)
		assert.Equal(t, []string{"2026-07-24", "2026-07-23"}, got.Dates)
	})

	t.Run("limit<=0なら既定値を使う", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ListAsOfDates(gomock.Any(), gomock.Eq(dailyAvoidStockDatesDefaultLimit)).Return(nil, nil)

		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		got, err := NewDailyAvoidStockInteractor(avoidRepo, brandRepo).GetAsOfDates(context.Background(), 0)
		assert.NoError(t, err)
		assert.NotNil(t, got.Dates, "nilではなく空スライス")
		assert.Empty(t, got.Dates)
	})
}

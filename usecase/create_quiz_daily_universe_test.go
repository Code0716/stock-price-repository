package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestCreateQuizDailyUniverseInteractorImpl_CreateQuizDailyUniverse(t *testing.T) {
	now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	type fields struct {
		priceRepo    func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		universeRepo func(ctrl *gomock.Controller) repositories.QuizDailyUniverseRepository
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "20営業日に満たない場合は何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, quizUniverseWindowDays).
						Return([]time.Time{now}, nil)
					return mock
				},
				universeRepo: func(ctrl *gomock.Controller) repositories.QuizDailyUniverseRepository {
					// ExistsByQuizDate/BulkCreate は呼ばれない
					return mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
				},
			},
		},
		{
			name: "既に当日分が作成済みなら何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					dates := make([]time.Time, quizUniverseWindowDays)
					for i := range dates {
						dates[i] = now.AddDate(0, 0, -i)
					}
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, quizUniverseWindowDays).Return(dates, nil)
					return mock
				},
				universeRepo: func(ctrl *gomock.Controller) repositories.QuizDailyUniverseRepository {
					mock := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
					mock.EXPECT().ExistsByQuizDate(gomock.Any(), now).Return(true, nil)
					return mock
				},
			},
		},
		{
			name: "正常系: 選定結果をBulkCreateする",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					dates := make([]time.Time, quizUniverseWindowDays)
					for i := range dates {
						dates[i] = now.AddDate(0, 0, -i)
					}
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, quizUniverseWindowDays).Return(dates, nil)
					mock.EXPECT().ListPricesByDateRange(gomock.Any(), dates[len(dates)-1], now).Return([]*models.StockBrandDailyPrice{
						{
							StockBrandID: "brand-a",
							TickerSymbol: "A001",
							Date:         now,
							Close:        decimal.NewFromInt(100),
							High:         decimal.NewFromInt(105),
							Low:          decimal.NewFromInt(95),
							Volume:       1000,
						},
					}, nil)
					return mock
				},
				universeRepo: func(ctrl *gomock.Controller) repositories.QuizDailyUniverseRepository {
					mock := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
					mock.EXPECT().ExistsByQuizDate(gomock.Any(), now).Return(false, nil)
					mock.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, entries []*models.QuizUniverseEntry) error {
							assert.Len(t, entries, 1)
							assert.Equal(t, "brand-a", entries[0].StockBrandID)
							return nil
						})
					return mock
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewCreateQuizDailyUniverseInteractor(tt.fields.priceRepo(ctrl), tt.fields.universeRepo(ctrl))
			err := interactor.CreateQuizDailyUniverse(context.Background(), now)
			assert.NoError(t, err)
		})
	}
}

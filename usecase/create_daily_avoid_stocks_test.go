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
	"github.com/Code0716/stock-price-repository/repositories"
)

func avoidTestDates(now time.Time) []time.Time {
	dates := make([]time.Time, dailyAvoidStockWindowDays)
	for i := range dates {
		dates[i] = now.AddDate(0, 0, -i)
	}
	return dates
}

// makeAvoidUsecasePrices n本のうち末尾だけ大きく動く日足系列を生成する（domain_service側のテストヘルパーと同趣旨）。
func makeAvoidUsecasePrices(n int, flatClose, lastClose decimal.Decimal) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, n)
	for i := 0; i < n-1; i++ {
		out[i] = &models.StockBrandDailyPrice{
			Date:     base.AddDate(0, 0, i),
			Close:    flatClose,
			High:     flatClose.Add(decimal.NewFromInt(1)),
			Low:      flatClose.Sub(decimal.NewFromInt(1)),
			Volume:   2_000_000,
			Adjclose: flatClose,
		}
	}
	out[n-1] = &models.StockBrandDailyPrice{
		Date:     base.AddDate(0, 0, n-1),
		Close:    lastClose,
		High:     lastClose.Add(decimal.NewFromInt(5)),
		Low:      lastClose.Sub(decimal.NewFromInt(5)),
		Volume:   6_000_000,
		Adjclose: lastClose,
	}
	return out
}

func TestCreateDailyAvoidStocksInteractorImpl_CreateDailyAvoidStocks(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	type fields struct {
		priceRepo func(ctrl *gomock.Controller) repositories.AdjustedDailyPriceRepository
		brandRepo func(ctrl *gomock.Controller) repositories.StockBrandRepository
		avoidRepo func(ctrl *gomock.Controller) repositories.DailyAvoidStockRepository
		tx        func(ctrl *gomock.Controller) repositories.Transaction
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "営業日数がボラ判定の最低日数未満なら何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.AdjustedDailyPriceRepository {
					mock := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyAvoidStockWindowDays).
						Return([]time.Time{now}, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				avoidRepo: func(ctrl *gomock.Controller) repositories.DailyAvoidStockRepository {
					return mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "asOfが指定されていればnowではなくasOfを基準にする",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.AdjustedDailyPriceRepository {
					mock := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
					asOf := now.AddDate(0, 0, -3)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), asOf, dailyAvoidStockWindowDays).
						Return([]time.Time{asOf}, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				avoidRepo: func(ctrl *gomock.Controller) repositories.DailyAvoidStockRepository {
					return mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "force=falseかつ当日分が既にあれば何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.AdjustedDailyPriceRepository {
					mock := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
					dates := avoidTestDates(now)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyAvoidStockWindowDays).Return(dates, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					// FindAllMainMarkets は呼ばれない（既存があるため再スクリーニングしない）
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				avoidRepo: func(ctrl *gomock.Controller) repositories.DailyAvoidStockRepository {
					mock := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
					mock.EXPECT().ExistsByAsOfDate(gomock.Any(), gomock.Any()).Return(true, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "正常系: 候補0件ならBulkCreateを呼ばない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.AdjustedDailyPriceRepository {
					mock := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
					dates := avoidTestDates(now)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyAvoidStockWindowDays).Return(dates, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{}, nil)
					return mock
				},
				avoidRepo: func(ctrl *gomock.Controller) repositories.DailyAvoidStockRepository {
					mock := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
					mock.EXPECT().ExistsByAsOfDate(gomock.Any(), gomock.Any()).Return(false, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewCreateDailyAvoidStocksInteractor(
				tt.fields.tx(ctrl),
				tt.fields.priceRepo(ctrl),
				tt.fields.brandRepo(ctrl),
				tt.fields.avoidRepo(ctrl),
			)
			var asOf *time.Time
			if tt.name == "asOfが指定されていればnowではなくasOfを基準にする" {
				d := now.AddDate(0, 0, -3)
				asOf = &d
			}
			err := interactor.CreateDailyAvoidStocks(context.Background(), now, asOf, 1, false)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDailyAvoidStocksInteractorImpl_CreateDailyAvoidStocks_Save(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	dates := avoidTestDates(now)
	asOfDate := dates[0]
	from := dates[len(dates)-1]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	priceRepo := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
	priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyAvoidStockWindowDays).Return(dates, nil)

	brand := &models.StockBrand{ID: "brand-1", TickerSymbol: "1000", Name: "テスト銘柄"}
	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{brand}, nil)

	prices := makeAvoidUsecasePrices(dailyAvoidStockWindowDays+1, decimal.NewFromInt(1000), decimal.NewFromInt(2000))
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
		TickerSymbol: "1000",
		DateFrom:     &from,
		DateTo:       &asOfDate,
		DateOrder:    dateOrderPtr(models.SortOrderAsc),
	}).Return(prices, nil)

	avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
	avoidRepo.EXPECT().ExistsByAsOfDate(gomock.Any(), gomock.Eq(asOfDate)).Return(false, nil)

	gomock.InOrder(
		avoidRepo.EXPECT().DeleteByAsOfDate(gomock.Any(), asOfDate).Return(nil),
		avoidRepo.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, rows []*models.DailyAvoidStock) error {
				assert.Len(t, rows, 1)
				assert.Equal(t, "1000", rows[0].TickerSymbol)
				assert.Equal(t, domain_service.DailyAvoidRuleVersion, rows[0].RuleVersion)
				return nil
			}),
	)

	tx := mock_repositories.NewMockTransaction(ctrl)
	tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	interactor := NewCreateDailyAvoidStocksInteractor(tx, priceRepo, brandRepo, avoidRepo)
	err := interactor.CreateDailyAvoidStocks(context.Background(), now, nil, 1, false)
	assert.NoError(t, err)
}

func TestCreateDailyAvoidStocksInteractorImpl_CreateDailyAvoidStocks_ForceRebuildsEvenIfExists(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	dates := avoidTestDates(now)
	asOfDate := dates[0]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	priceRepo := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
	priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyAvoidStockWindowDays).Return(dates, nil)
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{}, nil)

	avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
	// force=true なので ExistsByAsOfDate は呼ばれない
	_ = asOfDate

	tx := mock_repositories.NewMockTransaction(ctrl)

	interactor := NewCreateDailyAvoidStocksInteractor(tx, priceRepo, brandRepo, avoidRepo)
	err := interactor.CreateDailyAvoidStocks(context.Background(), now, nil, 1, true)
	assert.NoError(t, err, "候補0件ならBulkCreateまで到達せず正常終了する")
}

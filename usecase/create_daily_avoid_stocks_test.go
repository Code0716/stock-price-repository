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
	return avoidTestDatesN(now, dailyAvoidStockWindowDays)
}

// avoidTestDatesN now を起点に新しい順で n 件の日付を生成する。
func avoidTestDatesN(now time.Time, n int) []time.Time {
	dates := make([]time.Time, n)
	for i := range dates {
		dates[i] = now.AddDate(0, 0, -i)
	}
	return dates
}

// makeAvoidBackfillPrices dates（新しい順）の各日に対応する日足系列を昇順で生成する。
// BackfillDailyAvoidStocks は対象日ごとに異なる部分列を切り出すため、日付が dates と
// 1対1で一致している必要があり、makeAvoidUsecasePrices（固定baseの連番）は使えない。
func makeAvoidBackfillPrices(dates []time.Time, close decimal.Decimal, volume int64) []*models.StockBrandDailyPrice {
	out := make([]*models.StockBrandDailyPrice, len(dates))
	for i, d := range dates {
		out[len(dates)-1-i] = &models.StockBrandDailyPrice{
			Date:     d,
			Close:    close,
			High:     close.Add(decimal.NewFromInt(1)),
			Low:      close.Sub(decimal.NewFromInt(1)),
			Volume:   volume,
			Adjclose: close,
		}
	}
	return out
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

func TestCreateDailyAvoidStocksInteractorImpl_BackfillDailyAvoidStocks(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("days<=0はエラー", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		interactor := NewCreateDailyAvoidStocksInteractor(
			mock_repositories.NewMockTransaction(ctrl),
			mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl),
			mock_repositories.NewMockStockBrandRepository(ctrl),
			mock_repositories.NewMockDailyAvoidStockRepository(ctrl),
		)
		err := interactor.BackfillDailyAvoidStocks(context.Background(), now, 0, 1, false)
		assert.Error(t, err)
	})

	t.Run("営業日数がボラ判定の最低日数未満なら何もしない", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		priceRepo := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, 3+dailyAvoidStockWindowDays-1).
			Return([]time.Time{now}, nil)

		// FindAllMainMarkets は呼ばれない（最低日数未満で早期return）
		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

		interactor := NewCreateDailyAvoidStocksInteractor(
			mock_repositories.NewMockTransaction(ctrl),
			priceRepo,
			brandRepo,
			mock_repositories.NewMockDailyAvoidStockRepository(ctrl),
		)
		err := interactor.BackfillDailyAvoidStocks(context.Background(), now, 3, 1, false)
		assert.NoError(t, err)
	})

	t.Run("正常系: 価格取得は銘柄ごとに1回だけで複数日ぶん保存される", func(t *testing.T) {
		days := 3
		totalDates := days + dailyAvoidStockWindowDays - 1
		dates := avoidTestDatesN(now, totalDates)
		targetDates := dates[:days]
		from := dates[len(dates)-1]
		to := dates[0]

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		priceRepo := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, totalDates).Return(dates, nil)

		brand := &models.StockBrand{ID: "brand-1", TickerSymbol: "1000", Name: "テスト銘柄"}
		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{brand}, nil)

		prices := makeAvoidBackfillPrices(dates, decimal.NewFromInt(1000), 2_000_000)
		// 本テストの肝: days=3 に対して ListDailyPricesBySymbol は銘柄ごとに1回だけ呼ばれること
		// （重複取得を避けるのが BackfillDailyAvoidStocks を分離した目的）。
		priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
			TickerSymbol: "1000",
			DateFrom:     &from,
			DateTo:       &to,
			DateOrder:    dateOrderPtr(models.SortOrderAsc),
		}).Return(prices, nil).Times(1)

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ExistsByAsOfDate(gomock.Any(), gomock.Any()).Return(false, nil).Times(days)
		avoidRepo.EXPECT().DeleteByAsOfDate(gomock.Any(), gomock.Any()).Return(nil).Times(days)

		savedDates := make(map[time.Time]bool)
		avoidRepo.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, rows []*models.DailyAvoidStock) error {
				assert.Len(t, rows, 1)
				assert.Equal(t, "1000", rows[0].TickerSymbol)
				savedDates[rows[0].AsOfDate] = true
				return nil
			}).Times(days)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Times(days)

		interactor := NewCreateDailyAvoidStocksInteractor(tx, priceRepo, brandRepo, avoidRepo)
		err := interactor.BackfillDailyAvoidStocks(context.Background(), now, days, 1, false)
		assert.NoError(t, err)
		for _, d := range targetDates {
			assert.True(t, savedDates[d], "対象日 %s が保存されていること", d.Format("2006-01-02"))
		}
	})

	t.Run("1日分の保存に失敗しても残りの日は続行し、失敗日をまとめてエラーにする", func(t *testing.T) {
		days := 2
		totalDates := days + dailyAvoidStockWindowDays - 1
		dates := avoidTestDatesN(now, totalDates)
		from := dates[len(dates)-1]
		to := dates[0]

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		priceRepo := mock_repositories.NewMockAdjustedDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, totalDates).Return(dates, nil)

		brand := &models.StockBrand{ID: "brand-1", TickerSymbol: "1000", Name: "テスト銘柄"}
		brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{brand}, nil)

		prices := makeAvoidBackfillPrices(dates, decimal.NewFromInt(1000), 2_000_000)
		priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
			TickerSymbol: "1000",
			DateFrom:     &from,
			DateTo:       &to,
			DateOrder:    dateOrderPtr(models.SortOrderAsc),
		}).Return(prices, nil)

		avoidRepo := mock_repositories.NewMockDailyAvoidStockRepository(ctrl)
		avoidRepo.EXPECT().ExistsByAsOfDate(gomock.Any(), gomock.Any()).Return(false, nil).Times(days)
		avoidRepo.EXPECT().DeleteByAsOfDate(gomock.Any(), gomock.Any()).Return(nil).Times(days)

		callCount := 0
		avoidRepo.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, rows []*models.DailyAvoidStock) error {
				callCount++
				if callCount == 1 {
					return assert.AnError
				}
				return nil
			}).Times(days)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Times(days)

		interactor := NewCreateDailyAvoidStocksInteractor(tx, priceRepo, brandRepo, avoidRepo)
		err := interactor.BackfillDailyAvoidStocks(context.Background(), now, days, 1, false)
		assert.Error(t, err, "1日でも失敗すれば全体としてはエラーを返す（他の日は続行済み）")
	})
}

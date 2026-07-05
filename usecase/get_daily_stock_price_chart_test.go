package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/shopspring/decimal"
)

// buildDailyPricesForChartTest baseDate から days 日分の連続する日足データを生成する（Close は 100+i）。
func buildDailyPricesForChartTest(baseDate time.Time, days int) []*models.StockBrandDailyPrice {
	prices := make([]*models.StockBrandDailyPrice, days)
	for i := 0; i < days; i++ {
		prices[i] = &models.StockBrandDailyPrice{
			Date:   baseDate.AddDate(0, 0, i),
			Open:   decimal.NewFromInt(int64(100 + i)),
			High:   decimal.NewFromInt(int64(110 + i)),
			Low:    decimal.NewFromInt(int64(90 + i)),
			Close:  decimal.NewFromInt(int64(100 + i)),
			Volume: 1000,
		}
	}
	return prices
}

func TestStockBrandsDailyStockPriceInteractorImpl_GetDailyStockPriceChart(t *testing.T) {
	// from を指定した場合、リポジトリへの取得開始日は from の5ヶ月前になる
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	expectedFetchFrom := from.AddDate(0, -5, 0) // 2024-01-01

	// expectedFetchFrom から to までの連続する日足（161日分）。可視範囲（from以降）は10日分。
	daysFromFetchFromToTo := int(to.Sub(expectedFetchFrom).Hours()/24) + 1
	prices := buildDailyPricesForChartTest(expectedFetchFrom, daysFromFetchFromToTo)
	ascOrder := models.SortOrderAsc

	type fields struct {
		stockBrandsDailyStockPriceRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
	}
	type args struct {
		ctx    context.Context
		symbol string
		from   *time.Time
		to     *time.Time
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantCandles int
		wantErr     bool
	}{
		{
			name: "正常系: fromを指定した場合、5ヶ月前からウォームアップ取得しvisibleFrom以前は除外される",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
						TickerSymbol: "1234",
						DateFrom:     &expectedFetchFrom,
						DateTo:       &to,
						DateOrder:    &ascOrder,
					}).Return(prices, nil)
					return m
				},
			},
			args: args{
				ctx:    context.Background(),
				symbol: "1234",
				from:   &from,
				to:     &to,
			},
			wantCandles: 10, // from(2024-06-01)からto(2024-06-10)までの10日分のみ可視
			wantErr:     false,
		},
		{
			name: "正常系: fromがnilの場合は全期間取得しすべての点を可視とする",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
						TickerSymbol: "1234",
						DateFrom:     nil,
						DateTo:       nil,
						DateOrder:    &ascOrder,
					}).Return(buildDailyPricesForChartTest(from, 5), nil)
					return m
				},
			},
			args: args{
				ctx:    context.Background(),
				symbol: "1234",
				from:   nil,
				to:     nil,
			},
			wantCandles: 5,
			wantErr:     false,
		},
		{
			name: "異常系: リポジトリがエラーを返す場合はエラーを伝播する",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
			},
			args: args{
				ctx:    context.Background(),
				symbol: "1234",
				from:   &from,
				to:     &to,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := tt.fields.stockBrandsDailyStockPriceRepository(ctrl)
			u := NewStockBrandsDailyPriceInteractor(nil, nil, r, nil, nil, nil, nil)

			got, err := u.GetDailyStockPriceChart(tt.args.ctx, tt.args.symbol, tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDailyStockPriceChart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.Nil(t, got)
				return
			}

			assert.Len(t, got.Candles, tt.wantCandles)
			// MA75まで計算できるだけのウォームアップ期間があること
			if tt.name == "正常系: fromを指定した場合、5ヶ月前からウォームアップ取得しvisibleFrom以前は除外される" {
				assert.Len(t, got.MA75, tt.wantCandles)
			}
		})
	}
}

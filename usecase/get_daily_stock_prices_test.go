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

func TestStockBrandsDailyStockPriceInteractorImpl_GetDailyStockPrices(t *testing.T) {
	type fields struct {
		stockBrandsDailyStockPriceRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
	}
	type args struct {
		ctx    context.Context
		symbol string
		from   *time.Time
		to     *time.Time
	}

	now := time.Now()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*models.StockBrandDailyPrice
		wantErr bool
	}{
		{
			name: "正常系: 日足データを取得できる",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
						TickerSymbol: "1234",
						DateFrom:     &now,
						DateTo:       &now,
					}).Return([]*models.StockBrandDailyPrice{
						{
							StockBrandID: "1",
							Date:         now,
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil)
					return m
				},
			},
			args: args{
				ctx:    context.Background(),
				symbol: "1234",
				from:   &now,
				to:     &now,
			},
			want: []*models.StockBrandDailyPrice{
				{
					StockBrandID: "1",
					Date:         now,
					Open:         decimal.NewFromInt(100),
					High:         decimal.NewFromInt(110),
					Low:          decimal.NewFromInt(90),
					Close:        decimal.NewFromInt(105),
					Volume:       1000,
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: リポジトリがエラーを返す",
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
				from:   &now,
				to:     &now,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := tt.fields.stockBrandsDailyStockPriceRepository(ctrl)

			// 他の依存関係はnilでよい（GetDailyStockPricesでは使われないため）
			u := NewStockBrandsDailyPriceInteractor(nil, nil, r, nil, nil, nil, nil)

			got, err := u.GetDailyStockPrices(tt.args.ctx, tt.args.symbol, tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockBrandsDailyStockPriceInteractorImpl.GetDailyStockPrices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

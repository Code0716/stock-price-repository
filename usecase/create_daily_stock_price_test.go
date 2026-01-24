package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func Test_stockBrandsDailyStockPriceInteractorImpl_CreateDailyStockPrice(t *testing.T) {
	type fields struct {
		tx                                        func(ctrl *gomock.Controller) repositories.Transaction
		stockBrandRepository                      func(ctrl *gomock.Controller) repositories.StockBrandRepository
		stockBrandsDailyStockPriceRepository      func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		stockBrandsDailyPriceForAnalyzeRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		stockAPIClient                            func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	type args struct {
		ctx context.Context
		now time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					// CreateDailyStockPrice内で5回ループしてcreateDailyStockPriceを呼ぶため、5回DoInTxが呼ばれる
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					}).Times(5)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAll(gomock.Any()).Return([]*models.StockBrand{
						{
							ID:           "brand1",
							TickerSymbol: "1001",
						},
					}, nil).Times(5)
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(nil).Times(5)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).Times(5)
					mock.EXPECT().DeleteBeforeDate(gomock.Any(), gomock.Any()).Return(nil).Times(5)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol:    "1001",
							Date:            time.Now(),
							Open:            decimal.NewFromInt(100),
							High:            decimal.NewFromInt(110),
							Low:             decimal.NewFromInt(90),
							Close:           decimal.NewFromInt(105),
							Volume:          1000,
							AdjustmentClose: decimal.NewFromInt(105),
						},
					}, nil).Times(5)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "異常系: createDailyStockPriceでエラー (FindAllエラー)",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					// DoInTxが呼ばれるが、内部でエラーが返る
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					}).Times(1) // 最初のエラーで止まるので1回
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					// エラーを返す
					mock.EXPECT().FindAll(gomock.Any()).Return(nil, errors.New("db error")).Times(1)
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					// 呼ばれない
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					// 呼ばれない
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					// 呼ばれない (DoInTxの前に呼ばれるわけではない、DoInTx内の最初の処理でエラーなので呼ばれないかも？)
					// 実装を見ると newStockBrandDailyPrices は FindAll の後に呼ばれている。
					// 従って FindAll でエラーなら呼ばれない。
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			si := NewStockBrandsDailyPriceInteractor(
				tt.fields.tx(ctrl),
				tt.fields.stockBrandRepository(ctrl),
				tt.fields.stockBrandsDailyStockPriceRepository(ctrl),
				tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl),
				tt.fields.stockAPIClient(ctrl),
				nil, // redisClient (not used)
				nil, // slackAPIClient (not used)
			)

			if err := si.CreateDailyStockPrice(tt.args.ctx, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("stockBrandsDailyStockPriceInteractorImpl.CreateDailyStockPrice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

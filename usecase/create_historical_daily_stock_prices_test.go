package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func Test_stockBrandsDailyStockPriceInteractorImpl_CreateHistoricalDailyStockPrices(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	type fields struct {
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
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系: Redisにキーがない場合、最初から取得して保存する",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "0",
						Limit:           4000,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1001",
							Name:         "Test Brand",
						},
					}, nil)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gateway.StockAPISymbol("1001"), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							Date:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							TickerSymbol:    "1001",
							Open:            decimal.NewFromInt(100),
							High:            decimal.NewFromInt(110),
							Low:             decimal.NewFromInt(90),
							Close:           decimal.NewFromInt(105),
							Volume:          1000,
							AdjustmentClose: decimal.NewFromInt(105),
						},
					}, nil)
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() {
				s.FlushAll()
			},
			wantErr: false,
		},
		{
			name: "正常系: Redisにキーがある場合、続きから取得して保存する",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "1000",
						Limit:           4000,
					})).Return([]*models.StockBrand{
						{
							ID:           "2",
							TickerSymbol: "1002",
							Name:         "Test Brand 2",
						},
					}, nil)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gateway.StockAPISymbol("1002"), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							Date:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							TickerSymbol:    "1002",
							Open:            decimal.NewFromInt(200),
							High:            decimal.NewFromInt(210),
							Low:             decimal.NewFromInt(190),
							Close:           decimal.NewFromInt(205),
							Volume:          2000,
							AdjustmentClose: decimal.NewFromInt(205),
						},
					}, nil)
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() {
				s.FlushAll()
				s.Set(createHistoricalDailyStockPricesListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey, "1000")
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			si := &stockBrandsDailyStockPriceInteractorImpl{
				redisClient: redisClient,
			}

			if tt.fields.stockBrandRepository != nil {
				si.stockBrandRepository = tt.fields.stockBrandRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyStockPriceRepository != nil {
				si.stockBrandsDailyStockPriceRepository = tt.fields.stockBrandsDailyStockPriceRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceForAnalyzeRepository != nil {
				si.stockBrandsDailyPriceForAnalyzeRepository = tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl)
			}
			if tt.fields.stockAPIClient != nil {
				si.stockAPIClient = tt.fields.stockAPIClient(ctrl)
			}

			if err := si.CreateHistoricalDailyStockPrices(tt.args.ctx, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("stockBrandsDailyStockPriceInteractorImpl.CreateHistoricalDailyStockPrices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

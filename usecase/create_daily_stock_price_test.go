package usecase

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestStockBrandsDailyStockPriceInteractorImpl_CreateDailyStockPrice(t *testing.T) {
	// Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	// Check if redis is available
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

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
		wantErr bool
		setup   func(ctx context.Context)
		cleanup func(ctx context.Context)
	}{
		{
			name: "Success",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "0",
						Limit:           5000,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1111",
							Name:         "Test Company",
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
					// Verify DeleteBeforeDate is called
					mock.EXPECT().DeleteBeforeDate(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
			setup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
			cleanup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
		},
		{
			name: "Success - No stock brands",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "0",
						Limit:           5000,
					})).Return([]*models.StockBrand{}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
			setup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
			cleanup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
		},
		{
			name: "Error - FindWithFilter failed",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			setup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
			cleanup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
		},
		{
			name: "Error - createDailyStockPrice failed (CreateStockBrandDailyPrice error)",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "0",
						Limit:           5000,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1111",
							Name:         "Test Company",
						},
					}, nil)
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					// CreateStockBrandDailyPrice returns error
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			setup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
			cleanup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
		},
		{
			name: "Error - DeleteBeforeDate failed",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "0",
						Limit:           5000,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1111",
							Name:         "Test Company",
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
					// DeleteBeforeDate returns error
					mock.EXPECT().DeleteBeforeDate(gomock.Any(), gomock.Any()).Return(context.DeadlineExceeded)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			setup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
			cleanup: func(ctx context.Context) {
				redisClient.Del(ctx, createDailyStockPriceListToshyoStockBrandsBySymbolStockPriceRepositoryRedisKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup(tt.args.ctx)
			}
			if tt.cleanup != nil {
				defer tt.cleanup(tt.args.ctx)
			}

			s := &stockBrandsDailyStockPriceInteractorImpl{
				redisClient: redisClient,
			}
			if tt.fields.stockBrandRepository != nil {
				s.stockBrandRepository = tt.fields.stockBrandRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyStockPriceRepository != nil {
				s.stockBrandsDailyStockPriceRepository = tt.fields.stockBrandsDailyStockPriceRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceForAnalyzeRepository != nil {
				s.stockBrandsDailyPriceForAnalyzeRepository = tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl)
			}
			if tt.fields.stockAPIClient != nil {
				s.stockAPIClient = tt.fields.stockAPIClient(ctrl)
			}

			if err := s.CreateDailyStockPrice(tt.args.ctx, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("StockBrandsDailyStockPriceInteractorImpl.CreateDailyStockPrice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockBrandsDailyStockPriceInteractorImpl_createDailyStockPrice(t *testing.T) {
	type fields struct {
		stockBrandsDailyStockPriceRepository      func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		stockBrandsDailyPriceForAnalyzeRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		stockAPIClient                            func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	type args struct {
		ctx         context.Context
		stockBrands []*models.StockBrand
		now         time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success - Batch processing",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					// 101件なので、100件のバッチと1件のバッチで2回呼ばれる可能性があるが、
					// 並行処理の順序によってはバッチサイズに達する前に処理されることもあるため、
					// AnyTimes() で許容する。
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				stockBrands: func() []*models.StockBrand {
					brands := make([]*models.StockBrand, 101)
					for i := 0; i < 101; i++ {
						brands[i] = &models.StockBrand{
							ID:           fmt.Sprintf("%d", i),
							TickerSymbol: fmt.Sprintf("1%03d", i),
							Name:         "Test Company",
						}
					}
					return brands
				}(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "Error - CreateStockBrandDailyPrice failed",
			fields: fields{
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil).AnyTimes()
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				stockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1111",
						Name:         "Test Company",
					},
				},
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := &stockBrandsDailyStockPriceInteractorImpl{}
			if tt.fields.stockBrandsDailyStockPriceRepository != nil {
				s.stockBrandsDailyStockPriceRepository = tt.fields.stockBrandsDailyStockPriceRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceForAnalyzeRepository != nil {
				s.stockBrandsDailyPriceForAnalyzeRepository = tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl)
			}
			if tt.fields.stockAPIClient != nil {
				s.stockAPIClient = tt.fields.stockAPIClient(ctrl)
			}

			if err := s.createDailyStockPrice(tt.args.ctx, tt.args.stockBrands, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("StockBrandsDailyStockPriceInteractorImpl.createDailyStockPrice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockBrandsDailyStockPriceInteractorImpl_createDailyStockPriceBySymbol(t *testing.T) {
	type fields struct {
		stockAPIClient func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	type args struct {
		ctx         context.Context
		stockBrands []*models.StockBrand
		now         time.Time
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantPrices int
	}{
		{
			name: "Success",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{
							TickerSymbol: "1111",
							Date:         time.Now(),
							Open:         decimal.NewFromInt(100),
							High:         decimal.NewFromInt(110),
							Low:          decimal.NewFromInt(90),
							Close:        decimal.NewFromInt(105),
							Volume:       1000,
						},
					}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				stockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1111",
						Name:         "Test Company",
					},
				},
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantPrices: 1,
		},
		{
			name: "Error - API Error (Continue)",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				stockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1111",
						Name:         "Test Company",
					},
				},
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantPrices: 0,
		},
		{
			name: "Success - Empty Response",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetDailyPricesBySymbolAndRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				stockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1111",
						Name:         "Test Company",
					},
				},
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantPrices: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := &stockBrandsDailyStockPriceInteractorImpl{}
			if tt.fields.stockAPIClient != nil {
				s.stockAPIClient = tt.fields.stockAPIClient(ctrl)
			}

			stockBrandsCh := make(chan *models.StockBrand, len(tt.args.stockBrands))
			stockBrandDailyPricesCh := make(chan []*models.StockBrandDailyPrice, len(tt.args.stockBrands))
			var wg sync.WaitGroup

			for _, v := range tt.args.stockBrands {
				stockBrandsCh <- v
			}
			close(stockBrandsCh)

			wg.Add(1)
			go s.createDailyStockPriceBySymbol(tt.args.ctx, &wg, stockBrandsCh, stockBrandDailyPricesCh, tt.args.now)

			wg.Wait()
			close(stockBrandDailyPricesCh)

			count := 0
			for prices := range stockBrandDailyPricesCh {
				count += len(prices)
			}

			assert.Equal(t, tt.wantPrices, count)
		})
	}
}

func TestStockBrandsDailyStockPriceInteractorImpl_newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo(t *testing.T) {
	type args struct {
		stockBrand *models.StockBrand
		prices     []*gateway.StockPrice
		now        time.Time
	}
	tests := []struct {
		name string
		args args
		want []*models.StockBrandDailyPrice
	}{
		{
			name: "Success",
			args: args{
				stockBrand: &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1111",
					Name:         "Test Company",
				},
				prices: []*gateway.StockPrice{
					{
						TickerSymbol:    "1111",
						Date:            time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
						Open:            decimal.NewFromInt(100),
						High:            decimal.NewFromInt(110),
						Low:             decimal.NewFromInt(90),
						Close:           decimal.NewFromInt(105),
						Volume:          1000,
						AdjustmentClose: decimal.NewFromInt(105),
					},
				},
				now: time.Date(2023, 10, 2, 12, 0, 0, 0, time.UTC),
			},
			want: []*models.StockBrandDailyPrice{
				{
					StockBrandID: "1",
					TickerSymbol: "1111",
					Date:         time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
					Open:         decimal.NewFromInt(100),
					High:         decimal.NewFromInt(110),
					Low:          decimal.NewFromInt(90),
					Close:        decimal.NewFromInt(105),
					Volume:       1000,
					Adjclose:     decimal.NewFromInt(105),
				},
			},
		},
		{
			name: "Success - Empty Prices",
			args: args{
				stockBrand: &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1111",
					Name:         "Test Company",
				},
				prices: []*gateway.StockPrice{},
				now:    time.Date(2023, 10, 2, 12, 0, 0, 0, time.UTC),
			},
			want: nil,
		},
		{
			name: "Success - Nil Prices",
			args: args{
				stockBrand: &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1111",
					Name:         "Test Company",
				},
				prices: nil,
				now:    time.Date(2023, 10, 2, 12, 0, 0, 0, time.UTC),
			},
			want: nil,
		},
		{
			name: "Success - Skip Zero Values",
			args: args{
				stockBrand: &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1111",
					Name:         "Test Company",
				},
				prices: []*gateway.StockPrice{
					{
						TickerSymbol: "1111",
						Date:         time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
						// All zero values
					},
				},
				now: time.Date(2023, 10, 2, 12, 0, 0, 0, time.UTC),
			},
			want: []*models.StockBrandDailyPrice{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stockBrandsDailyStockPriceInteractorImpl{}
			got := s.newStockBrandDailyPriceByStockChartWithRangeAPIResponseInfo(tt.args.stockBrand, tt.args.prices, tt.args.now)

			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Len(t, got, len(tt.want))
				for i, wantItem := range tt.want {
					gotItem := got[i]
					// ID, CreatedAt, UpdatedAt are generated inside, so we check other fields
					assert.Equal(t, wantItem.StockBrandID, gotItem.StockBrandID)
					assert.Equal(t, wantItem.TickerSymbol, gotItem.TickerSymbol)
					assert.Equal(t, wantItem.Date, gotItem.Date)
					assert.Equal(t, wantItem.Open, gotItem.Open)
					assert.Equal(t, wantItem.High, gotItem.High)
					assert.Equal(t, wantItem.Low, gotItem.Low)
					assert.Equal(t, wantItem.Close, gotItem.Close)
					assert.Equal(t, wantItem.Volume, gotItem.Volume)
					assert.Equal(t, wantItem.Adjclose, gotItem.Adjclose)
				}
			}
		})
	}
}

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_database "github.com/Code0716/stock-price-repository/mock/database"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func Test_indexInteractorImpl_CreateNikkeiAndDjiHistoricalData(t *testing.T) {
	type fields struct {
		tx               func(ctrl *gomock.Controller) *mock_database.MockTransaction
		nikkeiRepository func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository
		djiRepository    func(ctrl *gomock.Controller) *mock_repositories.MockDjiRepository
		stockAPIClient   func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient
	}
	type args struct {
		ctx context.Context
		t   time.Time
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
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					// Nikkei
					mock.EXPECT().GetIndexPriceChart(
						gomock.Any(),
						gateway.StockAPISymbolNikkei,
						gateway.StockAPIInterval1D,
						gateway.StockAPIValidRange10Y,
					).Return(&gateway.StockChartWithRangeAPIResponseInfo{
						Indicator: []*gateway.StockPrice{
							{
								Date:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								Open:            decimal.NewFromInt(26000),
								High:            decimal.NewFromInt(26500),
								Low:             decimal.NewFromInt(25900),
								Close:           decimal.NewFromInt(26400),
								Volume:          1000000,
								AdjustmentClose: decimal.NewFromInt(26400),
							},
						},
					}, nil)
					// DJI
					mock.EXPECT().GetIndexPriceChart(
						gomock.Any(),
						gateway.StockAPISymbolDji,
						gateway.StockAPIInterval1D,
						gateway.StockAPIValidRange10Y,
					).Return(&gateway.StockChartWithRangeAPIResponseInfo{
						Indicator: []*gateway.StockPrice{
							{
								Date:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								Open:            decimal.NewFromInt(33000),
								High:            decimal.NewFromInt(33500),
								Low:             decimal.NewFromInt(32900),
								Close:           decimal.NewFromInt(33400),
								Volume:          2000000,
								AdjustmentClose: decimal.NewFromInt(33400),
							},
						},
					}, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) *mock_database.MockTransaction {
					mock := mock_database.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
					return mock
				},
				nikkeiRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					mock := mock_repositories.NewMockNikkeiRepository(ctrl)
					mock.EXPECT().CreateNikkeiStockAverageDailyPrices(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, prices models.IndexStockAverageDailyPrices) error {
						if len(prices) != 1 {
							return errors.New("unexpected length")
						}
						if !prices[0].Close.Equal(decimal.NewFromInt(26400)) {
							return errors.New("unexpected price")
						}
						return nil
					})
					return mock
				},
				djiRepository: func(ctrl *gomock.Controller) *mock_repositories.MockDjiRepository {
					mock := mock_repositories.NewMockDjiRepository(ctrl)
					mock.EXPECT().CreateDjiStockAverageDailyPrices(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, prices models.IndexStockAverageDailyPrices) error {
						if len(prices) != 1 {
							return errors.New("unexpected length")
						}
						if !prices[0].Close.Equal(decimal.NewFromInt(33400)) {
							return errors.New("unexpected price")
						}
						return nil
					})
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "異常系: 日経平均取得エラー",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetIndexPriceChart(
						gomock.Any(),
						gateway.StockAPISymbolNikkei,
						gateway.StockAPIInterval1D,
						gateway.StockAPIValidRange10Y,
					).Return(nil, errors.New("api error"))
					return mock
				},
				tx:               nil,
				nikkeiRepository: nil,
				djiRepository:    nil,
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "異常系: NYダウ取得エラー",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetIndexPriceChart(
						gomock.Any(),
						gateway.StockAPISymbolNikkei,
						gateway.StockAPIInterval1D,
						gateway.StockAPIValidRange10Y,
					).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					mock.EXPECT().GetIndexPriceChart(
						gomock.Any(),
						gateway.StockAPISymbolDji,
						gateway.StockAPIInterval1D,
						gateway.StockAPIValidRange10Y,
					).Return(nil, errors.New("api error"))
					return mock
				},
				tx:               nil,
				nikkeiRepository: nil,
				djiRepository:    nil,
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "異常系: トランザクションエラー",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolNikkei, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolDji, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) *mock_database.MockTransaction {
					mock := mock_database.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).Return(errors.New("tx error"))
					return mock
				},
				nikkeiRepository: nil,
				djiRepository:    nil,
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "異常系: 日経平均保存エラー(トランザクション内)",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolNikkei, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolDji, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) *mock_database.MockTransaction {
					mock := mock_database.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
					return mock
				},
				nikkeiRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					mock := mock_repositories.NewMockNikkeiRepository(ctrl)
					mock.EXPECT().CreateNikkeiStockAverageDailyPrices(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
					return mock
				},
				djiRepository: nil,
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "異常系: NYダウ保存エラー(トランザクション内)",
			fields: fields{
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolNikkei, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					mock.EXPECT().GetIndexPriceChart(gomock.Any(), gateway.StockAPISymbolDji, gomock.Any(), gomock.Any()).Return(&gateway.StockChartWithRangeAPIResponseInfo{}, nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) *mock_database.MockTransaction {
					mock := mock_database.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
					return mock
				},
				nikkeiRepository: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					mock := mock_repositories.NewMockNikkeiRepository(ctrl)
					mock.EXPECT().CreateNikkeiStockAverageDailyPrices(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				djiRepository: func(ctrl *gomock.Controller) *mock_repositories.MockDjiRepository {
					mock := mock_repositories.NewMockDjiRepository(ctrl)
					mock.EXPECT().CreateDjiStockAverageDailyPrices(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				t:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ii := &indexInteractorImpl{
				stockAPIClient: tt.fields.stockAPIClient(ctrl),
			}
			if tt.fields.tx != nil {
				ii.tx = tt.fields.tx(ctrl)
			}
			if tt.fields.nikkeiRepository != nil {
				ii.nikkeiRepository = tt.fields.nikkeiRepository(ctrl)
			}
			if tt.fields.djiRepository != nil {
				ii.djiRepository = tt.fields.djiRepository(ctrl)
			}

			if err := ii.CreateNikkeiAndDjiHistoricalData(tt.args.ctx, tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("indexInteractorImpl.CreateNikkeiAndDjiHistoricalData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_indexInteractorImpl_apiResponseToModel(t *testing.T) {
	type args struct {
		info *gateway.StockChartWithRangeAPIResponseInfo
		t    time.Time
	}
	tests := []struct {
		name string
		args args
		want models.IndexStockAverageDailyPrices
	}{
		{
			name: "正常系",
			args: args{
				info: &gateway.StockChartWithRangeAPIResponseInfo{
					Indicator: []*gateway.StockPrice{
						{
							Date:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							Open:            decimal.NewFromInt(100),
							High:            decimal.NewFromInt(110),
							Low:             decimal.NewFromInt(90),
							Close:           decimal.NewFromInt(105),
							Volume:          1000,
							AdjustmentClose: decimal.NewFromInt(105),
						},
					},
				},
				t: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			want: models.IndexStockAverageDailyPrices{
				{
					Date:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					Open:      decimal.NewFromInt(100),
					High:      decimal.NewFromInt(110),
					Low:       decimal.NewFromInt(90),
					Close:     decimal.NewFromInt(105),
					Volume:    1000,
					Adjclose:  decimal.NewFromInt(105),
					CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "正常系: 空のレスポンス",
			args: args{
				info: &gateway.StockChartWithRangeAPIResponseInfo{
					Indicator: []*gateway.StockPrice{},
				},
				t: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ii := &indexInteractorImpl{}
			got := ii.apiResponseToModel(tt.args.info, tt.args.t)

			if len(got) != len(tt.want) {
				t.Errorf("indexInteractorImpl.apiResponseToModel() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if !got[i].Date.Equal(tt.want[i].Date) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Date = %v, want %v", got[i].Date, tt.want[i].Date)
				}
				if !got[i].Open.Equal(tt.want[i].Open) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Open = %v, want %v", got[i].Open, tt.want[i].Open)
				}
				if !got[i].High.Equal(tt.want[i].High) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() High = %v, want %v", got[i].High, tt.want[i].High)
				}
				if !got[i].Low.Equal(tt.want[i].Low) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Low = %v, want %v", got[i].Low, tt.want[i].Low)
				}
				if !got[i].Close.Equal(tt.want[i].Close) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Close = %v, want %v", got[i].Close, tt.want[i].Close)
				}
				if got[i].Volume != tt.want[i].Volume {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Volume = %v, want %v", got[i].Volume, tt.want[i].Volume)
				}
				if !got[i].Adjclose.Equal(tt.want[i].Adjclose) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() Adjclose = %v, want %v", got[i].Adjclose, tt.want[i].Adjclose)
				}
				if !got[i].CreatedAt.Equal(tt.want[i].CreatedAt) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() CreatedAt = %v, want %v", got[i].CreatedAt, tt.want[i].CreatedAt)
				}
				if !got[i].UpdatedAt.Equal(tt.want[i].UpdatedAt) {
					t.Errorf("indexInteractorImpl.apiResponseToModel() UpdatedAt = %v, want %v", got[i].UpdatedAt, tt.want[i].UpdatedAt)
				}
			}
		})
	}
}

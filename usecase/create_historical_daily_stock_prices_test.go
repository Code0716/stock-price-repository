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
			name: "正常系: チェックポイントなし、全銘柄の日足を日付ループで取得して保存する",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{ID: "1", TickerSymbol: "1301", Name: "極洋"},
					}, nil)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					// 日付ループで呼ばれる。1月4日（水）のみデータを返す
					mock.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC)).Return([]*gateway.StockPrice{
						{
							Date:            time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC),
							TickerSymbol:    "1301",
							Open:            decimal.NewFromInt(3000),
							High:            decimal.NewFromInt(3100),
							Low:             decimal.NewFromInt(2900),
							Close:           decimal.NewFromInt(3050),
							Volume:          10000,
							AdjustmentClose: decimal.NewFromInt(3050),
						},
					}, nil).AnyTimes()
					// それ以外の日は空（祝日扱い）
					mock.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
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
				// 1月4日（水）のみ処理対象になるよう直前にチェックポイントを設定するケース
				now: time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC),
			},
			setup: func() {
				s.FlushAll()
				// 1月3日をチェックポイントにセット → 1月4日から開始
				s.Set(createHistoricalDailyStockPricesDateCheckpointRedisKey, "2023-01-03")
			},
			wantErr: false,
		},
		{
			name: "正常系: チェックポイントあり、続きの日付から取得して保存する",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{ID: "2", TickerSymbol: "1302", Name: "テスト銘柄2"},
					}, nil)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
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
				now: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), // 木曜
			},
			setup: func() {
				s.FlushAll()
				// 1月4日をチェックポイント → 1月5日から開始
				s.Set(createHistoricalDailyStockPricesDateCheckpointRedisKey, "2023-01-04")
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
				t.Errorf("CreateHistoricalDailyStockPrices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newStockBrandDailyPriceForAnalyzeFromGatewayPrices(t *testing.T) {
	now := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	date := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		prices []*gateway.StockPrice
		want   []*models.StockBrandDailyPriceForAnalyze
	}{
		{
			name: "Adjustment*が設定されている場合はそちらを採用する（分割調整済み値を保存する）",
			prices: []*gateway.StockPrice{
				{
					Date:             date,
					TickerSymbol:     "7678",
					Open:             decimal.NewFromInt(6330),
					High:             decimal.NewFromInt(6350),
					Low:              decimal.NewFromInt(6260),
					Close:            decimal.NewFromInt(6280),
					Volume:           23700,
					AdjustmentOpen:   decimal.NewFromInt(3165),
					AdjustmentHigh:   decimal.NewFromInt(3175),
					AdjustmentLow:    decimal.NewFromInt(3130),
					AdjustmentClose:  decimal.NewFromInt(3140),
					AdjustmentVolume: decimal.NewFromInt(47400),
				},
			},
			want: []*models.StockBrandDailyPriceForAnalyze{
				{
					TickerSymbol: "7678",
					Date:         date,
					Open:         decimal.NewFromInt(3165),
					High:         decimal.NewFromInt(3175),
					Low:          decimal.NewFromInt(3130),
					Close:        decimal.NewFromInt(3140),
					Volume:       47400,
					// close と adj_close は常に同値にする（二重調整を避けるため）
					Adjclose:  decimal.NewFromInt(3140),
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		{
			name: "Adjustment*が未設定(ゼロ値)の場合は素値にフォールバックする",
			prices: []*gateway.StockPrice{
				{
					Date:         date,
					TickerSymbol: "1301",
					Open:         decimal.NewFromInt(3000),
					High:         decimal.NewFromInt(3100),
					Low:          decimal.NewFromInt(2900),
					Close:        decimal.NewFromInt(3050),
					Volume:       10000,
				},
			},
			want: []*models.StockBrandDailyPriceForAnalyze{
				{
					TickerSymbol: "1301",
					Date:         date,
					Open:         decimal.NewFromInt(3000),
					High:         decimal.NewFromInt(3100),
					Low:          decimal.NewFromInt(2900),
					Close:        decimal.NewFromInt(3050),
					Volume:       10000,
					Adjclose:     decimal.NewFromInt(3050),
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
		},
		{
			name: "OHLC全てゼロの銘柄(売買なし)はスキップする",
			prices: []*gateway.StockPrice{
				{
					Date:         date,
					TickerSymbol: "9999",
					Open:         decimal.Zero,
					High:         decimal.Zero,
					Low:          decimal.Zero,
					Close:        decimal.Zero,
					Volume:       0,
				},
			},
			want: nil,
		},
		{
			name:   "空スライスはnilを返す",
			prices: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newStockBrandDailyPriceForAnalyzeFromGatewayPrices(tt.prices, now)
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("newStockBrandDailyPriceForAnalyzeFromGatewayPrices() = %v, want empty", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("newStockBrandDailyPriceForAnalyzeFromGatewayPrices() length = %d, want %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				g := got[i]
				if g.ID == "" {
					t.Errorf("expected non-empty ID")
				}
				if g.TickerSymbol != w.TickerSymbol ||
					!g.Date.Equal(w.Date) ||
					!g.Open.Equal(w.Open) ||
					!g.High.Equal(w.High) ||
					!g.Low.Equal(w.Low) ||
					!g.Close.Equal(w.Close) ||
					g.Volume != w.Volume ||
					!g.Adjclose.Equal(w.Adjclose) ||
					!g.Close.Equal(g.Adjclose) { // 不変条件: close == adj_close
					t.Errorf("newStockBrandDailyPriceForAnalyzeFromGatewayPrices()[%d] = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

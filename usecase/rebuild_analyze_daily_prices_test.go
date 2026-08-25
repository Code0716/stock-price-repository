package usecase

import (
	"context"
	"errors"
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

func Test_filterGatewayPricesBySymbols(t *testing.T) {
	prices := []*gateway.StockPrice{
		{TickerSymbol: "1001"},
		{TickerSymbol: "9999"},
		nil,
		{TickerSymbol: "1002"},
	}
	allowed := map[string]struct{}{"1001": {}, "1002": {}}

	got := filterGatewayPricesBySymbols(prices, allowed)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].TickerSymbol != "1001" || got[1].TickerSymbol != "1002" {
		t.Errorf("unexpected filtered symbols: %v", got)
	}
}

func Test_stockBrandsDailyStockPriceInteractorImpl_RebuildAnalyzeDailyPrices(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})

	type fields struct {
		stockBrandRepository                        func(ctrl *gomock.Controller) repositories.StockBrandRepository
		stockBrandsDailyPriceForAnalyzeRepository   func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		appliedStockSplitsHistoryRepository         func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository
		appliedStockConsolidationsHistoryRepository func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository
		stockAPIClient                              func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	tests := []struct {
		name    string
		fields  fields
		setup   func()
		now     time.Time
		wantErr bool
	}{
		{
			name: "チェックポイント無し: TRUNCATEしてから当日分を取得・保存する",
			now:  time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), // 月曜
			setup: func() {
				s.FlushAll()
			},
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{ID: "1", TickerSymbol: "1301"},
					}, nil)
					return m
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					m.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return m
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					m := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					return m
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					m := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					// 1301のみ返す。9999はmainMarketに含まれないので後でフィルタされる想定だが、
					// GetAllBrandDailyPricesByDate自体はJ-Quants側で全銘柄返すため両方投入する。
					m.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return([]*gateway.StockPrice{
						{TickerSymbol: "1301", Date: time.Now(), Open: decimal.NewFromInt(100), High: decimal.NewFromInt(110), Low: decimal.NewFromInt(90), Close: decimal.NewFromInt(105), Volume: 1000},
						{TickerSymbol: "9999", Date: time.Now(), Open: decimal.NewFromInt(1), High: decimal.NewFromInt(1), Low: decimal.NewFromInt(1), Close: decimal.NewFromInt(1), Volume: 1},
					}, nil).AnyTimes()
					return m
				},
			},
			wantErr: false,
		},
		{
			name: "チェックポイントあり: TRUNCATEをスキップし続きの日付から再開する",
			now:  time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC),
			setup: func() {
				s.FlushAll()
				s.Set(rebuildAnalyzeDailyPricesDateCheckpointRedisKey, "2026-08-21")
			},
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{ID: "1", TickerSymbol: "1301"},
					}, nil)
					return m
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					// TruncateAllは呼ばれない
					m.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
					return m
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					return mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl) // 呼ばれない
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl) // 呼ばれない
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
					return m
				},
			},
			wantErr: false,
		},
		{
			name: "API障害時はエラーを返さずチェックポイントを保存して終了する",
			now:  time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC),
			setup: func() {
				s.FlushAll()
			},
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{ID: "1", TickerSymbol: "1301"},
					}, nil)
					return m
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					return m
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					m := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					return m
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					m := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					m.EXPECT().TruncateAll(gomock.Any()).Return(nil).Times(1)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetAllBrandDailyPricesByDate(gomock.Any(), gomock.Any()).Return(nil, errors.New("api error")).AnyTimes()
					return m
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			si := &stockBrandsDailyStockPriceInteractorImpl{
				stockBrandRepository:                        tt.fields.stockBrandRepository(ctrl),
				stockBrandsDailyPriceForAnalyzeRepository:   tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl),
				appliedStockSplitsHistoryRepository:         tt.fields.appliedStockSplitsHistoryRepository(ctrl),
				appliedStockConsolidationsHistoryRepository: tt.fields.appliedStockConsolidationsHistoryRepository(ctrl),
				stockAPIClient:                              tt.fields.stockAPIClient(ctrl),
				redisClient:                                 redisClient,
			}

			if err := si.RebuildAnalyzeDailyPrices(context.Background(), tt.now); (err != nil) != tt.wantErr {
				t.Errorf("RebuildAnalyzeDailyPrices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

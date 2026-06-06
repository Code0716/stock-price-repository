package usecase

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestStockBrandInteractorImpl_SyncFinStatementsAllStocks(t *testing.T) {
	type fields struct {
		stockBrandRepo   func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandRepository
		stockAPIClient   func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient
		finStatementRepo func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository
		redisSetup       func(mr *miniredis.Miniredis)
	}
	type args struct {
		intervalMs int
		max        int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, mr *miniredis.Miniredis)
	}{
		{
			name: "(a) 正常: 全銘柄を処理してチェックポイントを削除する",
			fields: fields{
				stockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{TickerSymbol: "1301"},
						{TickerSymbol: "1302"},
					}, nil)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1301")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1302")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil)
					return m
				},
				finStatementRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)
					return m
				},
			},
			args: args{intervalMs: 0, max: 0},
			check: func(t *testing.T, mr *miniredis.Miniredis) {
				t.Helper()
				assert.False(t, mr.Exists(syncFinStatementsAllStocksCheckpointRedisKey), "完了後はチェックポイントキーが削除されているべき")
			},
		},
		{
			name: "(b) 1銘柄がAPIエラー → ログして継続、他銘柄は処理される",
			fields: fields{
				stockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{TickerSymbol: "1301"},
						{TickerSymbol: "1302"},
						{TickerSymbol: "1303"},
					}, nil)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1301")).
						Return(nil, errors.New("api error"))
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1302")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1303")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil)
					return m
				},
				finStatementRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)
					return m
				},
			},
			args:    args{intervalMs: 0, max: 0},
			wantErr: false,
		},
		{
			name: "(c) max=1: 1銘柄だけ処理して打ち切り",
			fields: fields{
				stockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{TickerSymbol: "1301"},
						{TickerSymbol: "1302"},
					}, nil)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1301")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil).Times(1)
					return m
				},
				finStatementRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					return m
				},
			},
			args: args{intervalMs: 0, max: 1},
		},
		{
			name: "(d) 再開: チェックポイントが中間symbolにセット済み → 以前の銘柄はスキップ",
			fields: fields{
				stockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{
						{TickerSymbol: "1301"},
						{TickerSymbol: "1302"},
						{TickerSymbol: "1303"},
					}, nil)
					return m
				},
				stockAPIClient: func(ctrl *gomock.Controller) *mock_gateway.MockStockAPIClient {
					m := mock_gateway.NewMockStockAPIClient(ctrl)
					m.EXPECT().GetFinancialStatementsBySymbol(gomock.Any(), gateway.StockAPISymbol("1303")).
						Return([]*gateway.FinancialStatementsResponseInfo{}, nil).Times(1)
					return m
				},
				finStatementRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					return m
				},
				redisSetup: func(mr *miniredis.Miniredis) {
					mr.Set(syncFinStatementsAllStocksCheckpointRedisKey, "1302")
				},
			},
			args: args{intervalMs: 0, max: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mr, redisClient := newTestRedis(t)
			if tt.fields.redisSetup != nil {
				tt.fields.redisSetup(mr)
			}

			si := &stockBrandInteractorImpl{
				stockBrandRepository:   tt.fields.stockBrandRepo(ctrl),
				stockAPIClient:         tt.fields.stockAPIClient(ctrl),
				finStatementRepository: tt.fields.finStatementRepo(ctrl),
				redisClient:            redisClient,
			}

			err := si.SyncFinStatementsAllStocks(context.Background(), tt.args.intervalMs, tt.args.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncFinStatementsAllStocks() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, mr)
			}
		})
	}
}

func TestReadSyncFinStatementsAllStocksCheckpoint(t *testing.T) {
	tests := []struct {
		name       string
		redisSetup func(mr *miniredis.Miniredis)
		want       string
		wantErr    bool
	}{
		{
			name:    "キーなし → 空文字を返す",
			want:    "",
			wantErr: false,
		},
		{
			name: "キーあり → checkpoint値を返す",
			redisSetup: func(mr *miniredis.Miniredis) {
				mr.Set(syncFinStatementsAllStocksCheckpointRedisKey, "7203")
			},
			want:    "7203",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, redisClient := newTestRedis(t)
			if tt.redisSetup != nil {
				tt.redisSetup(mr)
			}

			si := &stockBrandInteractorImpl{redisClient: redisClient}
			got, err := si.readSyncFinStatementsAllStocksCheckpoint(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("readSyncFinStatementsAllStocksCheckpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

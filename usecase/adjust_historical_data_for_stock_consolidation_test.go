package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestAdjustHistoricalDataForStockConsolidationInteractor_AdjustHistoricalDataForStockConsolidation(t *testing.T) {
	type fields struct {
		stockBrandsDailyPriceForAnalyzeRepository   func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		appliedStockConsolidationsHistoryRepository func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository
	}
	type args struct {
		ctx                context.Context
		code               string
		consolidationDate  time.Time
		consolidationRatio decimal.Decimal
		dryRun             bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系: 併合実行 (5:1併合)",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)

					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPriceForAnalyze{
						{
							TickerSymbol: "1001",
							Date:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							Open:         decimal.NewFromInt(1000),
							Close:        decimal.NewFromInt(1000),
							High:         decimal.NewFromInt(1000),
							Low:          decimal.NewFromInt(1000),
							Adjclose:     decimal.NewFromInt(1000),
							Volume:       100,
						},
					}, nil)
					// 価格は5倍 (1000 → 5000)、出来高は1/5 (100 → 20) になることを確認
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, prices []*models.StockBrandDailyPriceForAnalyze) error {
						if len(prices) != 1 {
							return errors.New("unexpected price length")
						}
						p := prices[0]
						expectedPrice := decimal.NewFromInt(5000)
						if !p.Open.Equal(expectedPrice) {
							return errors.New("unexpected open price")
						}
						expectedVolume := int64(20)
						if p.Volume != expectedVolume {
							return errors.New("unexpected volume")
						}
						return nil
					})
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: false,
		},
		{
			name: "正常系: 併合日が未来の場合はスキップ",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Now().Add(24 * time.Hour),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: false,
		},
		{
			name: "正常系: 適用済みのためスキップ",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(true, nil)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: false,
		},
		{
			name: "正常系: DryRun (保存処理が呼ばれない)",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPriceForAnalyze{
						{
							TickerSymbol: "1001",
							Date:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							Open:         decimal.NewFromInt(1000),
							Close:        decimal.NewFromInt(1000),
							High:         decimal.NewFromInt(1000),
							Low:          decimal.NewFromInt(1000),
							Adjclose:     decimal.NewFromInt(1000),
							Volume:       100,
						},
					}, nil)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             true,
			},
			wantErr: false,
		},
		{
			name: "正常系: 対象データなし",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPriceForAnalyze{}, nil)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: false,
		},
		{
			name: "異常系: データ取得エラー",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: true,
		},
		{
			name: "異常系: データ保存エラー",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPriceForAnalyze{
						{
							TickerSymbol: "1001",
							Date:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							Open:         decimal.NewFromInt(1000),
							Close:        decimal.NewFromInt(1000),
							High:         decimal.NewFromInt(1000),
							Low:          decimal.NewFromInt(1000),
							Adjclose:     decimal.NewFromInt(1000),
							Volume:       100,
						},
					}, nil)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(errors.New("db save error"))
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:                context.Background(),
				code:               "1001",
				consolidationDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				consolidationRatio: decimal.NewFromInt(5),
				dryRun:             false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ui := NewAdjustHistoricalDataForStockConsolidation(
				tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl),
				tt.fields.appliedStockConsolidationsHistoryRepository(ctrl),
			)
			if err := ui.AdjustHistoricalDataForStockConsolidation(tt.args.ctx, tt.args.code, tt.args.consolidationDate, tt.args.consolidationRatio, tt.args.dryRun); (err != nil) != tt.wantErr {
				t.Errorf("AdjustHistoricalDataForStockConsolidationInteractor.AdjustHistoricalDataForStockConsolidation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

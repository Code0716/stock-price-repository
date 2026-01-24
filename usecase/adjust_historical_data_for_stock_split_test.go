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

func TestAdjustHistoricalDataForStockSplitInteractor_AdjustHistoricalDataForStockSplit(t *testing.T) {
	type fields struct {
		stockBrandsDailyPriceForAnalyzeRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		appliedStockSplitsHistoryRepository       func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository
	}
	type args struct {
		ctx        context.Context
		code       string
		splitDate  time.Time
		splitRatio decimal.Decimal
		dryRun     bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系: 分割実行 (1:2分割)",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)

					// 株式分割履歴の存在確認
					// (このフィールドは fields に移動したため、ここでは設定せず、下の appliedStockSplitsHistoryRepository で設定します)

					// データ取得のモック
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
					// データ保存のモック (値が半分になっていることを確認、出来高は倍になっていることを確認)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, prices []*models.StockBrandDailyPriceForAnalyze) error {
						if len(prices) != 1 {
							return errors.New("unexpected price length")
						}
						p := prices[0]
						expectedPrice := decimal.NewFromInt(500)
						if !p.Open.Equal(expectedPrice) {
							return errors.New("unexpected open price")
						}
						expectedVolume := int64(200)
						if p.Volume != expectedVolume {
							return errors.New("unexpected volume")
						}
						return nil
					})
					return mock
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					// 未実行であることを返す
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					// 履歴登録が呼ばれることを確認
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     false,
			},
			wantErr: false,
		},
		{
			name: "正常系: 適用済みのためスキップ",
			fields: fields{
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					// 実行済みの場合はデータ取得も保存も呼ばれない
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					// 適用済みを返す
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(true, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     false,
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
					// 保存処理は呼ばれないはず
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					// DryRun の場合は履歴登録も呼ばれない
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     true,
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
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     false,
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
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     false,
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
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "1001", gomock.Any()).Return(false, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				code:       "1001",
				splitDate:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				splitRatio: decimal.NewFromInt(2),
				dryRun:     false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ui := NewAdjustHistoricalDataForStockSplit(
				tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl),
				tt.fields.appliedStockSplitsHistoryRepository(ctrl),
			)
			if err := ui.AdjustHistoricalDataForStockSplit(tt.args.ctx, tt.args.code, tt.args.splitDate, tt.args.splitRatio, tt.args.dryRun); (err != nil) != tt.wantErr {
				t.Errorf("AdjustHistoricalDataForStockSplitInteractor.AdjustHistoricalDataForStockSplit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

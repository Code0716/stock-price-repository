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

func TestStockBrandsDailyPriceInteractorImpl_AdjustHistoricalDataForStockSplit(t *testing.T) {
	type fields struct {
		tx                                   func(ctrl *gomock.Controller) repositories.Transaction
		stockBrandsDailyStockPriceRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
	}
	type args struct {
		ctx        context.Context
		symbol     string
		splitRatio decimal.Decimal
		effectiveDate   time.Time
		dryRun     bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系: 1:2分割 (価格半分、出来高倍)",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					// データ取得
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
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
					// 保存 (値が半分、出来高が倍になっているか確認)
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, prices []*models.StockBrandDailyPrice) error {
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
			},
			args: args{
				ctx:        context.Background(),
				symbol:     "1001",
				splitRatio: decimal.NewFromInt(2),
				effectiveDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				dryRun:     false,
			},
			wantErr: false,
		},
		{
			name: "正常系: 対象データなし",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					// データがない場合はTxは開始されない
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbol:     "1001",
				splitRatio: decimal.NewFromInt(2),
				effectiveDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				dryRun:     false,
			},
			wantErr: false,
		},
		{
			name: "正常系: DryRun (保存処理が呼ばれない)",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					// DryRunの場合はTxは開始されない
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
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
					// 保存は呼ばれない
					mock.EXPECT().CreateStockBrandDailyPrice(gomock.Any(), gomock.Any()).Times(0)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbol:     "1001",
				splitRatio: decimal.NewFromInt(2),
				effectiveDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				dryRun:     true,
			},
			wantErr: false,
		},
		{
			name: "異常系: データ取得エラー",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbol:     "1001",
				splitRatio: decimal.NewFromInt(2),
				effectiveDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				dryRun:     false,
			},
			wantErr: true,
		},
		{
			name: "異常系: 保存時トランザクションエラー",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).Return(errors.New("tx error"))
					return mock
				},
				stockBrandsDailyStockPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
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
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbol:     "1001",
				splitRatio: decimal.NewFromInt(2),
				effectiveDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				dryRun:     false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 他の依存関係はnilで良い（このテストでは使わない）
			ui := NewStockBrandsDailyPriceInteractor(
				tt.fields.tx(ctrl),
				nil,
				tt.fields.stockBrandsDailyStockPriceRepository(ctrl),
				nil,
				nil,
				nil,
				nil,
			)

			if err := ui.AdjustHistoricalDataForStockSplit(tt.args.ctx, tt.args.symbol, tt.args.splitRatio, tt.args.effectiveDate, tt.args.dryRun); (err != nil) != tt.wantErr {
				t.Errorf("StockBrandsDailyPriceInteractorImpl.AdjustHistoricalDataForStockSplit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

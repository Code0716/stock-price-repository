package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

func TestE2E_AdjustHistoricalDataForStockConsolidation(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T, analyzeRepo *database.StockBrandsDailyPriceForAnalyzeRepositoryImpl, brandRepo *database.StockBrandRepositoryImpl, mockSlackAPI *mock_gateway.MockSlackAPIClient)
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "正常系: 株式併合に伴う過去データの修正が成功する",
			args: args{
				cmdArgs: []string{
					"main", "adjust_historical_data_for_stock_consolidation",
					"--code=1001",
					"--consolidation-date=2023-10-03",
					"--consolidation-ratio=5.0",
				},
			},
			setup: func(t *testing.T, analyzeRepo *database.StockBrandsDailyPriceForAnalyzeRepositoryImpl, brandRepo *database.StockBrandRepositoryImpl, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				ctx := context.Background()

				mockSlackAPI.EXPECT().
					SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", nil).
					AnyTimes()

				brand := &models.StockBrand{
					ID:           "uuid-1001",
					TickerSymbol: "1001",
					Name:         "Test Brand",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := brandRepo.UpsertStockBrands(ctx, []*models.StockBrand{brand})
				require.NoError(t, err)

				// 過去データ（併合前）の作成
				// 日付: 2023-10-01, 2023-10-02
				// 価格: Open 100, Close 100, Volume 500
				date1 := time.Date(2023, 10, 1, 0, 0, 0, 0, time.Local)
				date2 := time.Date(2023, 10, 2, 0, 0, 0, 0, time.Local)

				prices := []*models.StockBrandDailyPriceForAnalyze{
					{
						ID:           util.GenerateUUID(),
						TickerSymbol: "1001",
						Date:         date1,
						Open:         decimal.NewFromInt(100),
						Close:        decimal.NewFromInt(100),
						High:         decimal.NewFromInt(100),
						Low:          decimal.NewFromInt(100),
						Adjclose:     decimal.NewFromInt(100),
						Volume:       500,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           util.GenerateUUID(),
						TickerSymbol: "1001",
						Date:         date2,
						Open:         decimal.NewFromInt(100),
						Close:        decimal.NewFromInt(100),
						High:         decimal.NewFromInt(100),
						Low:          decimal.NewFromInt(100),
						Adjclose:     decimal.NewFromInt(100),
						Volume:       500,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err = analyzeRepo.CreateStockBrandDailyPriceForAnalyze(ctx, prices)
				require.NoError(t, err)
			},
			wantErr: false,
			check: func(t *testing.T) {
				// 併合比率 5.0 なので、価格は ×5 (500), 出来高は ÷5 (100) になっているはず
				var prices []genModel.StockBrandsDailyPriceForAnalyze
				err := db.Where("ticker_symbol = ?", "1001").Order("date asc").Find(&prices).Error
				require.NoError(t, err)
				require.Len(t, prices, 2)

				assert.Equal(t, 500.0, prices[0].OpenPrice, "OpenPrice should be multiplied by ratio")
				assert.Equal(t, uint64(100), prices[0].Volume, "Volume should be divided by ratio")

				assert.Equal(t, 500.0, prices[1].OpenPrice, "OpenPrice should be multiplied by ratio")
				assert.Equal(t, uint64(100), prices[1].Volume, "Volume should be divided by ratio")

				// 履歴の検証
				var history genModel.AppliedStockConsolidationsHistory
				err = db.Where("symbol = ?", "1001").First(&history).Error
				require.NoError(t, err)
				assert.Equal(t, "1001", history.Symbol)

				expectedDate := time.Date(2023, 10, 3, 0, 0, 0, 0, time.Local)
				assert.Equal(t, expectedDate.Format("2006-01-02"), history.ConsolidationDate.Format("2006-01-02"))
				assert.Equal(t, 5.0, history.Ratio)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.TruncateAllTables(t, db)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)

			analyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db).(*database.StockBrandsDailyPriceForAnalyzeRepositoryImpl)
			brandRepo := database.NewStockBrandRepositoryImpl(db).(*database.StockBrandRepositoryImpl)
			appliedStockConsolidationsRepo := database.NewAppliedStockConsolidationsHistoryRepositoryImpl(db)

			if tt.setup != nil {
				tt.setup(t, analyzeRepo, brandRepo, mockSlackAPI)
			}

			interactor := usecase.NewAdjustHistoricalDataForStockConsolidation(
				analyzeRepo,
				appliedStockConsolidationsRepo,
			)

			cmd := commands.NewAdjustHistoricalDataForStockConsolidationCommand(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				AdjustHistoricalDataForStockConsolidationCommand: cmd,
				SlackAPIClient: mockSlackAPI,
			})

			err := runner.Run(context.Background(), tt.args.cmdArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

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

func TestE2E_AdjustHistoricalDataForStockSplit(t *testing.T) {
	// 1. Setup DB
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
			name: "正常系: 株式分割に伴う過去データの修正が成功する",
			args: args{
				cmdArgs: []string{
					"main", "adjust_historical_data_for_stock_split",
					"--code=1001",
					"--split-date=2023-10-03",
					"--split-ratio=2.0",
				},
			},
			setup: func(t *testing.T, analyzeRepo *database.StockBrandsDailyPriceForAnalyzeRepositoryImpl, brandRepo *database.StockBrandRepositoryImpl, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				ctx := context.Background()

				// Slack通知のモック設定
				mockSlackAPI.EXPECT().
					SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", nil).
					AnyTimes()

				// 銘柄の作成
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

				// 過去データ（分割前）の作成
				// 日付: 2023-10-01, 2023-10-02
				// 価格: Open 200, Close 200, Volume 100
				date1 := time.Date(2023, 10, 1, 0, 0, 0, 0, time.Local)
				date2 := time.Date(2023, 10, 2, 0, 0, 0, 0, time.Local)

				prices := []*models.StockBrandDailyPriceForAnalyze{
					{
						ID:           util.GenerateUUID(),
						TickerSymbol: "1001",
						Date:         date1,
						Open:         decimal.NewFromInt(200),
						Close:        decimal.NewFromInt(200),
						High:         decimal.NewFromInt(200),
						Low:         decimal.NewFromInt(200),
						Adjclose:     decimal.NewFromInt(200),
						Volume:       100,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           util.GenerateUUID(),
						TickerSymbol: "1001",
						Date:         date2,
						Open:         decimal.NewFromInt(200),
						Close:        decimal.NewFromInt(200),
						High:         decimal.NewFromInt(200),
						Low:         decimal.NewFromInt(200),
						Adjclose:     decimal.NewFromInt(200),
						Volume:       100,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err = analyzeRepo.CreateStockBrandDailyPriceForAnalyze(ctx, prices)
				require.NoError(t, err)
			},
			wantErr: false,
			check: func(t *testing.T) {
				// データの検証
				// 分割比率 2.0 なので、価格は 1/2 (100), 出来高は *2 (200) になっているはず

				var prices []genModel.StockBrandsDailyPriceForAnalyze
				err := db.Where("ticker_symbol = ?", "1001").Order("date asc").Find(&prices).Error
				require.NoError(t, err)
				require.Len(t, prices, 2)

				// 2023-10-01
				assert.Equal(t, 100.0, prices[0].OpenPrice, "OpenPrice should be halved")
				assert.Equal(t, uint64(200), prices[0].Volume, "Volume should be doubled")

				// 2023-10-02
				assert.Equal(t, 100.0, prices[1].OpenPrice, "OpenPrice should be halved")
				assert.Equal(t, uint64(200), prices[1].Volume, "Volume should be doubled")

				// 履歴の検証
				var history genModel.AppliedStockSplitsHistory
				err = db.Where("symbol = ?", "1001").First(&history).Error
				require.NoError(t, err)
				assert.Equal(t, "1001", history.Symbol)
				
				// 日付の比較（時刻部分は無視されることを期待、またはDBから取得した値がDATE型なら文字列比較などが安全かも）
				// genModelのSplitDateがtime.Time型の場合、DB保存時に時刻が0になっているはず
				expectedDate := time.Date(2023, 10, 3, 0, 0, 0, 0, time.Local)
				// MySQL driver might return time in Local or UTC depending on config, usually parseTime=true makes it UTC or Local.
				// 比較時は日付部分だけ比較するのが無難だが、一旦Equalで試す。
				// setupTestDBの設定による。通常はUTCで扱うのがベストプラクティス。
				// ここでは YYYY-MM-DD 文字列で比較してみる。
				assert.Equal(t, expectedDate.Format("2006-01-02"), history.SplitDate.Format("2006-01-02"))
				
				assert.Equal(t, 2.0, history.Ratio)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.TruncateAllTables(t, db)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)

			// リポジトリの準備
			// 構造体へのキャストが必要なら修正する。ここではインターフェースではなく具象型を受け取るようにSetupを定義している。
			// しかしNew...関数はインターフェースを返すので、型アサーションが必要。
			analyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db).(*database.StockBrandsDailyPriceForAnalyzeRepositoryImpl)
			brandRepo := database.NewStockBrandRepositoryImpl(db).(*database.StockBrandRepositoryImpl)
			appliedStockSplitsRepo := database.NewAppliedStockSplitsHistoryRepositoryImpl(db)

			if tt.setup != nil {
				tt.setup(t, analyzeRepo, brandRepo, mockSlackAPI)
			}

			// Interactorの準備
			interactor := usecase.NewAdjustHistoricalDataForStockSplit(
				analyzeRepo,
				appliedStockSplitsRepo,
			)

			// Commandの準備
			cmd := commands.NewAdjustHistoricalDataForStockSplitCommand(interactor)

			// Runnerの準備
			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				AdjustHistoricalDataForStockSplitCommand: cmd,
				SlackAPIClient:                           mockSlackAPI,
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

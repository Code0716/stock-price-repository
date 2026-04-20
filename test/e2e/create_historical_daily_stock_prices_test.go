package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_CreateHistoricalDailyStockPrices(t *testing.T) {
	// 1. Setup DB
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// 2. Setup Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient)
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "正常系: 過去の日足データの作成が成功する",
			args: args{
				cmdArgs: []string{"main", "create_historical_daily_stock_price"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				// Prepare Data
				stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
				brand := &models.StockBrand{
					ID:           "1",
					TickerSymbol: "9999",
					Name:         "Historical Test Brand",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err = stockBrandRepo.UpsertStockBrands(context.Background(), []*models.StockBrand{brand})
				assert.NoError(t, err)

				// 直近1週間分のみ処理するようチェックポイントをセット
				checkpoint := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
				mr.Set("create_historical_daily_stock_prices_date_checkpoint", checkpoint)

				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

				// 日付ループで呼ばれる: 各平日について全銘柄の価格を返す
				mockStockAPI.EXPECT().GetAllBrandDailyPricesByDate(
					gomock.Any(),
					gomock.Any(),
				).DoAndReturn(func(_ context.Context, date time.Time) ([]*gateway.StockPrice, error) {
					return []*gateway.StockPrice{
						{
							Date:            date,
							TickerSymbol:    "9999",
							Open:            decimal.NewFromInt(1000),
							High:            decimal.NewFromInt(1100),
							Low:             decimal.NewFromInt(900),
							Close:           decimal.NewFromInt(1050),
							Volume:          5000,
							AdjustmentClose: decimal.NewFromInt(1050),
						},
					}, nil
				}).AnyTimes()
			},
			wantErr: false,
			check: func(t *testing.T) {
				var prices []*genModel.StockBrandsDailyPrice
				err = db.Where("stock_brand_id = ?", "1").Find(&prices).Error
				assert.NoError(t, err)
				assert.NotEmpty(t, prices, "9999の日足データが作成されているべき")
				assert.Equal(t, 1000.0, prices[0].OpenPrice)

				var analyzePrices []*genModel.StockBrandsDailyPriceForAnalyze
				err = db.Where("ticker_symbol = ?", "9999").Find(&analyzePrices).Error
				assert.NoError(t, err)
				assert.NotEmpty(t, analyzePrices, "9999のanalyzeデータが作成されているべき")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.TruncateAllTables(t, db)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
			mockStockAPI := mock_gateway.NewMockStockAPIClient(ctrl)

			if tt.setup != nil {
				tt.setup(t, mockStockAPI, mockSlackAPI)
			}

			stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
			dailyPriceRepo := database.NewStockBrandsDailyPriceRepositoryImpl(db)
			analyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
			tx := database.NewTransaction(db)

			interactor := usecase.NewStockBrandsDailyPriceInteractor(
				tx,
				stockBrandRepo,
				dailyPriceRepo,
				analyzeRepo,
				mockStockAPI,
				redisClient,
				mockSlackAPI,
			)

			cmd := commands.NewCreateHistoricalDailyStockPricesV1Command(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				CreateHistoricalDailyStockPricesV1Command: cmd,
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

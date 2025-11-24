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

func TestE2E_CreateDailyStockPrice(t *testing.T) {
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
			name: "正常系: 日足データの作成が成功する",
			args: args{
				cmdArgs: []string{"main", "create_daily_stock_price_v1"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				// Insert a stock brand
				stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
				brand := &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1234",
					Name:         "Test Brand",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err = stockBrandRepo.UpsertStockBrands(context.Background(), []*models.StockBrand{brand})
				assert.NoError(t, err)

				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

				mockStockAPI.EXPECT().GetDailyPricesBySymbolAndRange(
					gomock.Any(),
					gateway.StockAPISymbol("1234"),
					gomock.Any(), // from
					gomock.Any(), // to
				).DoAndReturn(func(_ context.Context, symbol gateway.StockAPISymbol, _, to time.Time) ([]*gateway.StockPrice, error) {
					// Return some dummy prices
					return []*gateway.StockPrice{
						{
							Date:            to,
							TickerSymbol:    string(symbol),
							Open:            decimal.NewFromInt(100),
							High:            decimal.NewFromInt(110),
							Low:             decimal.NewFromInt(90),
							Close:           decimal.NewFromInt(105),
							Volume:          1000,
							AdjustmentClose: decimal.NewFromInt(105),
						},
					}, nil
				})
			},
			wantErr: false,
			check: func(t *testing.T) {
				// Check if daily price was inserted
				var prices []*genModel.StockBrandsDailyPrice
				err = db.Where("stock_brand_id = ?", "1").Find(&prices).Error
				assert.NoError(t, err)
				assert.Len(t, prices, 1)
				assert.Equal(t, "1", prices[0].StockBrandID)
				assert.Equal(t, "1234", prices[0].TickerSymbol)
				// genModel uses float64, so we compare with float
				assert.Equal(t, 100.0, prices[0].OpenPrice)

				// Check analyze repo as well
				var analyzePrices []*genModel.StockBrandsDailyPriceForAnalyze
				err = db.Where("ticker_symbol = ?", "1234").Find(&analyzePrices).Error
				assert.NoError(t, err)
				assert.Len(t, analyzePrices, 1)
				assert.Equal(t, "1234", analyzePrices[0].TickerSymbol)
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

			// 4. Setup Repositories
			stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
			dailyPriceRepo := database.NewStockBrandsDailyPriceRepositoryImpl(db)
			analyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)

			// 5. Setup Interactor
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

			// 6. Setup Command
			cmd := commands.NewCreateDailyStockPriceV1Command(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				CreateDailyStockPriceV1Command: cmd,
				SlackAPIClient:                 mockSlackAPI,
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

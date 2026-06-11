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
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_CreateNikkeiAndDjiHistoricalData(t *testing.T) {
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
			name: "正常系: 日経平均・DJI・TOPIXの過去データ作成が成功する",
			args: args{
				cmdArgs: []string{"main", "create_nikkei_and_dji_historical_data_v1"},
			},
			setup: func(_ *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				// Setup Expectations
				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

				// Nikkei
				mockStockAPI.EXPECT().GetIndexPriceChart(
					gomock.Any(),
					gateway.StockAPISymbolNikkei,
					gateway.StockAPIInterval1D,
					gateway.StockAPIValidRange10Y,
				).Return(&gateway.StockChartWithRangeAPIResponseInfo{
					TickerSymbol: string(gateway.StockAPISymbolNikkei),
					Indicator: []*gateway.StockPrice{
						{
							Date:            time.Now().AddDate(0, 0, -1),
							TickerSymbol:    string(gateway.StockAPISymbolNikkei),
							Open:            decimal.NewFromInt(30000),
							High:            decimal.NewFromInt(30500),
							Low:             decimal.NewFromInt(29500),
							Close:           decimal.NewFromInt(30100),
							Volume:          100000,
							AdjustmentClose: decimal.NewFromInt(30100),
						},
					},
				}, nil)

				// DJI
				mockStockAPI.EXPECT().GetIndexPriceChart(
					gomock.Any(),
					gateway.StockAPISymbolDji,
					gateway.StockAPIInterval1D,
					gateway.StockAPIValidRange10Y,
				).Return(&gateway.StockChartWithRangeAPIResponseInfo{
					TickerSymbol: string(gateway.StockAPISymbolDji),
					Indicator: []*gateway.StockPrice{
						{
							Date:            time.Now().AddDate(0, 0, -1),
							TickerSymbol:    string(gateway.StockAPISymbolDji),
							Open:            decimal.NewFromInt(35000),
							High:            decimal.NewFromInt(35500),
							Low:             decimal.NewFromInt(34500),
							Close:           decimal.NewFromInt(35100),
							Volume:          200000,
							AdjustmentClose: decimal.NewFromInt(35100),
						},
					},
				}, nil)

				// TOPIX ETF (1306.T) を TOPIX 代理として使用
				mockStockAPI.EXPECT().GetIndexPriceChart(
					gomock.Any(),
					gateway.StockAPISymbolTopixETF,
					gateway.StockAPIInterval1D,
					gateway.StockAPIValidRange10Y,
				).Return(&gateway.StockChartWithRangeAPIResponseInfo{
					TickerSymbol: string(gateway.StockAPISymbolTopixETF),
					Indicator: []*gateway.StockPrice{
						{
							Date:            time.Now().AddDate(0, 0, -1),
							TickerSymbol:    string(gateway.StockAPISymbolTopixETF),
							Open:            decimal.NewFromInt(1900),
							High:            decimal.NewFromInt(1950),
							Low:             decimal.NewFromInt(1880),
							Close:           decimal.NewFromInt(1940),
							Volume:          500000,
							AdjustmentClose: decimal.NewFromInt(1940),
						},
					},
				}, nil)
			},
			wantErr: false,
			check: func(t *testing.T) {
				// Verify Nikkei Results
				var nikkeiPrices []*genModel.NikkeiStockAverageDailyPrice
				err = db.Find(&nikkeiPrices).Error
				assert.NoError(t, err)
				assert.Len(t, nikkeiPrices, 1)
				assert.Equal(t, 30000.0, nikkeiPrices[0].OpenPrice)

				var djiPrices []*genModel.DjiStockAverageDailyStockPrice
				err = db.Find(&djiPrices).Error
				assert.NoError(t, err)
				assert.Len(t, djiPrices, 1)
				assert.Equal(t, 35000.0, djiPrices[0].OpenPrice)

				// Verify TOPIX Results
				var topixPrices []*genModel.TopixDailyPrice
				err = db.Find(&topixPrices).Error
				assert.NoError(t, err)
				assert.Len(t, topixPrices, 1)
				assert.Equal(t, 1900.0, topixPrices[0].OpenPrice)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.TruncateAllTables(t, db)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStockAPI := mock_gateway.NewMockStockAPIClient(ctrl)
			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)

			if tt.setup != nil {
				tt.setup(t, mockStockAPI, mockSlackAPI)
			}

			// 4. Setup Repositories
			djiRepo := database.NewDjiRepositoryImpl(db)
			nikkeiRepo := database.NewNikkeiRepositoryImpl(db)
			topixRepo := database.NewTopixRepositoryImpl(db)

			// 5. Setup Interactor
			tx := database.NewTransaction(db)
			interactor := usecase.NewIndexInteractor(tx, redisClient, nikkeiRepo, djiRepo, topixRepo, mockStockAPI, mockSlackAPI)

			// 6. Setup Command
			cmd := commands.NewCreateNikkeiAndDjiHistoricalDataV1Command(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				SlackAPIClient: mockSlackAPI,
				CreateNikkeiAndDjiHistoricalDataV1Command: cmd,
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

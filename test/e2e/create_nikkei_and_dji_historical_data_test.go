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

	// 3. Setup Mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStockAPI := mock_gateway.NewMockStockAPIClient(ctrl)
	mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)

	// 4. Setup Repositories
	djiRepo := database.NewDjiRepositoryImpl(db)
	nikkeiRepo := database.NewNikkeiRepositoryImpl(db)

	// 5. Setup Interactor
	tx := database.NewTransaction(db)
	interactor := usecase.NewIndexInteractor(tx, redisClient, nikkeiRepo, djiRepo, mockStockAPI, mockSlackAPI)

	// 6. Setup Command
	cmd := commands.NewCreateNikkeiAndDjiHistoricalDataV1Command(interactor)

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		SlackAPIClient: mockSlackAPI,
		CreateNikkeiAndDjiHistoricalDataV1Command: cmd,
	})

	// 7. Setup Expectations
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

	// 8. Run Command
	args := []string{"main", "create_nikkei_and_dji_historical_data_v1"}
	err = runner.Run(context.Background(), args)
	assert.NoError(t, err)

	// 9. Verify Results
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
}

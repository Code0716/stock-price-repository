package e2e

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

func TestE2E_UpdateStockBrands(t *testing.T) {
	// 1. Setup DB
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// 2. Setup Redis (Miniredis)
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

	// 4. Setup Dependencies
	// Repositories
	stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
	sbDailyRepo := database.NewStockBrandsDailyPriceRepositoryImpl(db)
	analyzeRepo := database.NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	sbDailyAnalyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
	tx := database.NewTransaction(db)

	// Interactor
	interactor := usecase.NewStockBrandInteractor(
		tx,
		stockBrandRepo,
		sbDailyRepo,
		analyzeRepo,
		sbDailyAnalyzeRepo,
		mockStockAPI,
		redisClient,
	)

	// Commands
	updateCmd := commands.NewUpdateStockBrandsV1Command(interactor)

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		UpdateStockBrandsV1Command: updateCmd,
		SlackAPIClient:             mockSlackAPI,
	})

	// 5. Define Test Data & Expectations
	expectedBrands := []*gateway.StockBrand{
		{
			Symbol:           "1001",
			CompanyName:      "Test Company",
			MarketCode:       "111",
			MarketCodeName:   "Prime",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery, Agriculture & Forestry",
			Sector17Code:     "1",
			Sector17CodeName: "Foods",
		},
	}

	// Mock Expectations
	mockStockAPI.EXPECT().GetStockBrands(gomock.Any()).Return(expectedBrands, nil)

	// Slack expectations
	mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil)

	// 6. Run Command
	args := []string{"main", "update_stock_brands_v1"}
	err = runner.Run(context.Background(), args)
	assert.NoError(t, err)

	// 7. Verify DB
	var count int64
	db.Model(&genModel.StockBrand{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var brand genModel.StockBrand
	db.First(&brand)
	assert.Equal(t, "1001", brand.TickerSymbol)
	assert.Equal(t, "Test Company", brand.Name)
}

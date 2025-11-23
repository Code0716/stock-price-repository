package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_ExportSQL(t *testing.T) {
	// 1. Setup Mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
	mockMySQLDump := mock_gateway.NewMockMySQLDumpClient(ctrl)

	// 2. Setup Interactor
	interactor := usecase.NewExportSQLInteractor(mockMySQLDump)

	// 3. Setup Command
	cmd := commands.NewExportStockBrandsAndDailyPriceToSQLV1Command(interactor)

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		ExportStockBrandsAndDailyPriceToSQLV1Command: cmd,
		SlackAPIClient: mockSlackAPI,
	})

	// 4. Setup Expectations
	mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

	// Expect ExportTableAll for specific tables
	tables := []string{
		gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
		gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice,
		gateway.MySQLDumpTableNameStockBrand,
		gateway.MySQLDumpTableNameSector33AverageDailyPrice,
	}

	for _, table := range tables {
		mockMySQLDump.EXPECT().ExportTableAll(
			gomock.Any(),
			gomock.Any(), // filename (contains date)
			table,
		).Return(nil)
	}

	// Expect ExportDailyStockPriceByYear for years from 2019 to now
	now := time.Now()
	for year := 2019; year <= now.Year(); year++ {
		mockMySQLDump.EXPECT().ExportDailyStockPriceByYear(
			gomock.Any(),
			year,
		).Return(nil)
	}

	// 5. Run Command
	args := []string{"main", "export_stock_brands_daily_stock_price_to_sql_v1"}
	err := runner.Run(context.Background(), args)
	assert.NoError(t, err)
}

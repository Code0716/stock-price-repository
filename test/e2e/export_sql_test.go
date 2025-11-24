package e2e

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_ExportSQL(t *testing.T) {
	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T, mockMySQLDump *mock_gateway.MockMySQLDumpClient, mockSlackAPI *mock_gateway.MockSlackAPIClient)
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "正常系: SQLエクスポートが成功する",
			args: args{
				cmdArgs: []string{"main", "export_stock_brands_daily_stock_price_to_sql_v1"},
			},
			setup: func(_ *testing.T, mockMySQLDump *mock_gateway.MockMySQLDumpClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				// Setup Expectations
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
			},
			wantErr: false,
			check:   func(_ *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
			mockMySQLDump := mock_gateway.NewMockMySQLDumpClient(ctrl)

			if tt.setup != nil {
				tt.setup(t, mockMySQLDump, mockSlackAPI)
			}

			// 2. Setup Interactor
			interactor := usecase.NewExportSQLInteractor(mockMySQLDump)

			// 3. Setup Command
			cmd := commands.NewExportStockBrandsAndDailyPriceToSQLV1Command(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				ExportStockBrandsAndDailyPriceToSQLV1Command: cmd,
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

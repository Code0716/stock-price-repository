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
			name: "正常系: 銘柄情報の更新が成功する",
			args: args{
				cmdArgs: []string{"main", "update_stock_brands_v1"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
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
				mockSlackAPI.EXPECT().SendMessageByStrings(
					gomock.Any(),
					gateway.SlackChannelNameDevNotification,
					gomock.Any(),
					nil,
					nil,
				).DoAndReturn(func(ctx context.Context, channelName gateway.SlackChannelName, title string, message, ts *string) (string, error) {
					assert.Contains(t, title, "command name: update_stock_brands_v1")
					return "", nil
				})
			},
			wantErr: false,
			check: func(t *testing.T) {
				var count int64
				db.Model(&genModel.StockBrand{}).Count(&count)
				assert.Equal(t, int64(1), count)

				var brand genModel.StockBrand
				db.First(&brand)
				assert.Equal(t, "1001", brand.TickerSymbol)
				assert.Equal(t, "Test Company", brand.Name)
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

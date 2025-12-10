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

// TestE2E_MarketCodeFilter_CreateDailyStockPrice - CreateDailyStockPriceが主要市場（111, 112, 113）のみを対象とすることを検証
func TestE2E_MarketCodeFilter_CreateDailyStockPrice(t *testing.T) {
	// 1. Setup DB
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// 2. Setup Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer miniredis.Run()

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
			name: "主要市場（111, 112, 113）のみ日足データを取得する",
			args: args{
				cmdArgs: []string{"main", "create_daily_stock_price_v1"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
				now := time.Now()

				// 主要市場の銘柄を投入
				brands := []*models.StockBrand{
					{
						ID:           "prime-1",
						TickerSymbol: "1001",
						Name:         "Prime Brand 1",
						MarketCode:   "111", // Prime
						MarketName:   "Prime",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
					{
						ID:           "standard-1",
						TickerSymbol: "1002",
						Name:         "Standard Brand 1",
						MarketCode:   "112", // Standard
						MarketName:   "Standard",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
					{
						ID:           "growth-1",
						TickerSymbol: "1003",
						Name:         "Growth Brand 1",
						MarketCode:   "113", // Growth
						MarketName:   "Growth",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
					{
						ID:           "other-1",
						TickerSymbol: "1004",
						Name:         "Other Market Brand",
						MarketCode:   "999", // その他の市場
						MarketName:   "Other",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
				}
				err = stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				assert.NoError(t, err)

				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

				// 主要市場の3銘柄についてのみAPIコールが行われることを期待
				for _, symbol := range []string{"1001", "1002", "1003"} {
					mockStockAPI.EXPECT().GetDailyPricesBySymbolAndRange(
						gomock.Any(),
						gateway.StockAPISymbol(symbol),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(_ context.Context, sym gateway.StockAPISymbol, _, to time.Time) ([]*gateway.StockPrice, error) {
						return []*gateway.StockPrice{
							{
								Date:            to,
								TickerSymbol:    string(sym),
								Open:            decimal.NewFromInt(100),
								High:            decimal.NewFromInt(110),
								Low:             decimal.NewFromInt(90),
								Close:           decimal.NewFromInt(105),
								Volume:          1000,
								AdjustmentClose: decimal.NewFromInt(105),
							},
						}, nil
					})
				}

				// market_code="999" の銘柄についてはAPIコールされないことを期待（Timesを設定しない）
			},
			wantErr: false,
			check: func(t *testing.T) {
				// 主要市場の3銘柄について日足データが作成されていることを確認
				var prices []*genModel.StockBrandsDailyPrice
				err = db.Find(&prices).Error
				assert.NoError(t, err)
				assert.Len(t, prices, 3, "主要市場の3銘柄のみ日足データが作成されるべき")

				// その他市場の銘柄（1004）について日足データが作成されていないことを確認
				var otherMarketPrices []*genModel.StockBrandsDailyPrice
				err = db.Where("ticker_symbol = ?", "1004").Find(&otherMarketPrices).Error
				assert.NoError(t, err)
				assert.Len(t, otherMarketPrices, 0, "market_code=999の銘柄は日足データが作成されないべき")

				// 作成された日足データの ticker_symbol を確認
				actualSymbols := make([]string, 0, len(prices))
				for _, p := range prices {
					actualSymbols = append(actualSymbols, p.TickerSymbol)
				}
				assert.ElementsMatch(t, []string{"1001", "1002", "1003"}, actualSymbols)

				// analyze リポジトリも同様に主要市場のみであることを確認
				var analyzePrices []*genModel.StockBrandsDailyPriceForAnalyze
				err = db.Find(&analyzePrices).Error
				assert.NoError(t, err)
				assert.Len(t, analyzePrices, 3, "主要市場の3銘柄のみ分析用データが作成されるべき")
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

			// 7. Execute Command
			err = runner.Run(context.Background(), tt.args.cmdArgs)

			// 8. Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

// TestE2E_MarketCodeFilter_CreateHistoricalDailyStockPrices - CreateHistoricalDailyStockPricesが主要市場のみを対象とすることを検証
func TestE2E_MarketCodeFilter_CreateHistoricalDailyStockPrices(t *testing.T) {
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
			name: "主要市場（111, 112, 113）のみ過去の日足データを取得する",
			args: args{
				cmdArgs: []string{"main", "create_historical_daily_stock_price"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
				now := time.Now()

				// 主要市場の銘柄を投入
				brands := []*models.StockBrand{
					{
						ID:           "prime-1",
						TickerSymbol: "2001",
						Name:         "Prime Brand 2",
						MarketCode:   "111", // Prime
						MarketName:   "Prime",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
					{
						ID:           "standard-1",
						TickerSymbol: "2002",
						Name:         "Standard Brand 2",
						MarketCode:   "112", // Standard
						MarketName:   "Standard",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
					{
						ID:           "other-2",
						TickerSymbol: "2003",
						Name:         "Other Market Brand 2",
						MarketCode:   "888", // その他の市場
						MarketName:   "Other",
						CreatedAt:    now,
						UpdatedAt:    now,
					},
				}
				err = stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				assert.NoError(t, err)

				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()

				// 主要市場の2銘柄についてのみAPIコールが行われることを期待
				for _, symbol := range []string{"2001", "2002"} {
					mockStockAPI.EXPECT().GetDailyPricesBySymbolAndRange(
						gomock.Any(),
						gateway.StockAPISymbol(symbol),
						gomock.Any(),
						gomock.Any(),
					).DoAndReturn(func(_ context.Context, sym gateway.StockAPISymbol, from, to time.Time) ([]*gateway.StockPrice, error) {
						// 過去5年分のダミーデータを返す（簡略化のため3日分）
						prices := make([]*gateway.StockPrice, 0, 3)
						for i := 0; i < 3; i++ {
							date := to.AddDate(0, 0, -i)
							prices = append(prices, &gateway.StockPrice{
								Date:            date,
								TickerSymbol:    string(sym),
								Open:            decimal.NewFromInt(100 + int64(i)),
								High:            decimal.NewFromInt(110 + int64(i)),
								Low:             decimal.NewFromInt(90 + int64(i)),
								Close:           decimal.NewFromInt(105 + int64(i)),
								Volume:          1000,
								AdjustmentClose: decimal.NewFromInt(105 + int64(i)),
							})
						}
						return prices, nil
					})
				}

				// market_code="888" の銘柄についてはAPIコールされないことを期待
			},
			wantErr: false,
			check: func(t *testing.T) {
				// 主要市場の2銘柄について日足データが作成されていることを確認
				var prices []*genModel.StockBrandsDailyPrice
				err = db.Find(&prices).Error
				assert.NoError(t, err)
				// 2銘柄 × 3日分 = 6レコード
				assert.Len(t, prices, 6, "主要市場の2銘柄のみ日足データが作成されるべき")

				// その他市場の銘柄（2003）について日足データが作成されていないことを確認
				var otherMarketPrices []*genModel.StockBrandsDailyPrice
				err = db.Where("ticker_symbol = ?", "2003").Find(&otherMarketPrices).Error
				assert.NoError(t, err)
				assert.Len(t, otherMarketPrices, 0, "market_code=888の銘柄は日足データが作成されないべき")

				// 作成された日足データの ticker_symbol のユニーク値を確認
				var uniqueSymbols []string
				err = db.Model(&genModel.StockBrandsDailyPrice{}).
					Distinct("ticker_symbol").
					Pluck("ticker_symbol", &uniqueSymbols).Error
				assert.NoError(t, err)
				assert.ElementsMatch(t, []string{"2001", "2002"}, uniqueSymbols)
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
			cmd := commands.NewCreateHistoricalDailyStockPricesV1Command(interactor)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				CreateHistoricalDailyStockPricesV1Command: cmd,
				SlackAPIClient: mockSlackAPI,
			})

			// 7. Execute Command
			err = runner.Run(context.Background(), tt.args.cmdArgs)

			// 8. Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

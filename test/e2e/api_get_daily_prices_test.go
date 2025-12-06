package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	"github.com/Code0716/stock-price-repository/entrypoint/api/router"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_GetDailyPrices(t *testing.T) {
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

	// 3. Setup Dependencies
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
	mockStockAPI := mock_gateway.NewMockStockAPIClient(ctrl)

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

	httpServer := driver.NewHTTPServer()
	h := handler.NewStockPriceHandler(interactor, httpServer, zap.NewNop())
	mux := router.NewRouter(h)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 4. Test Cases
	tests := []struct {
		name       string
		setup      func(t *testing.T)
		query      string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name: "Success: Get prices by symbol",
			setup: func(t *testing.T) {
				// Insert StockBrand
				brand := &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1301",
					Name:         "Test Brand",
					MarketName:   "Prime",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), []*models.StockBrand{brand})
				require.NoError(t, err)

				// Insert test data
				prices := []*models.StockBrandDailyPrice{
					{
						ID:           "1",
						StockBrandID: "1",
						TickerSymbol: "1301",
						Date:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						Open:         decimal.NewFromInt(100),
						High:         decimal.NewFromInt(110),
						Low:          decimal.NewFromInt(90),
						Close:        decimal.NewFromInt(105),
						Volume:       1000,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "2",
						StockBrandID: "1",
						TickerSymbol: "1301",
						Date:         time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
						Open:         decimal.NewFromInt(105),
						High:         decimal.NewFromInt(115),
						Low:          decimal.NewFromInt(100),
						Close:        decimal.NewFromInt(110),
						Volume:       1200,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err = dailyPriceRepo.CreateStockBrandDailyPrice(context.Background(), prices)
				require.NoError(t, err)
			},
			query:      "?symbol=1301",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res []*models.StockBrandDailyPrice
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				require.Len(t, res, 2)
				assert.Equal(t, "1301", res[0].TickerSymbol)
			},
		},
		{
			name: "Success: Get prices with date range",
			setup: func(t *testing.T) {
				// Insert StockBrand (Upsert handles duplicates)
				brand := &models.StockBrand{
					ID:           "1",
					TickerSymbol: "1301",
					Name:         "Test Brand",
					MarketName:   "Prime",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), []*models.StockBrand{brand})
				require.NoError(t, err)

				prices := []*models.StockBrandDailyPrice{
					{
						ID:           "3",
						StockBrandID: "1",
						TickerSymbol: "1301",
						Date:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						Open:         decimal.NewFromInt(100),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "4",
						StockBrandID: "1",
						TickerSymbol: "1301",
						Date:         time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
						Open:         decimal.NewFromInt(120),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err = dailyPriceRepo.CreateStockBrandDailyPrice(context.Background(), prices)
				require.NoError(t, err)
			},
			query:      "?symbol=1301&from=2023-01-01&to=2023-01-03",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res []*models.StockBrandDailyPrice
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				require.Len(t, res, 1)
				assert.Equal(t, "1301", res[0].TickerSymbol)
				assert.Equal(t, "2023-01-01", res[0].Date.Format("2006-01-02"))
			},
		},
		{
			name:       "Error: Missing symbol",
			setup:      func(t *testing.T) {},
			query:      "",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "シンボルは必須です")
			},
		},
		{
			name:       "Error: Invalid date format",
			setup:      func(t *testing.T) {},
			query:      "?symbol=1301&from=invalid",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "fromの日付形式が不正です")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.TruncateAllTables(t, db)
			if tt.setup != nil {
				tt.setup(t)
			}

			res, err := http.Get(ts.URL + "/daily-prices" + tt.query)
			assert.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.check != nil {
				// Read body
				buf := new(bytes.Buffer)
				buf.ReadFrom(res.Body)
				tt.check(t, buf.Bytes())
			}
		})
	}
}

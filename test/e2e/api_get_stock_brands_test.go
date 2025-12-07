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

func TestE2E_GetStockBrands(t *testing.T) {
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
	analyzeRepo := database.NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	dailyPriceForAnalyzeRepo := database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
	tx := database.NewTransaction(db)

	stockBrandInteractor := usecase.NewStockBrandInteractor(
		tx,
		stockBrandRepo,
		dailyPriceRepo,
		analyzeRepo,
		dailyPriceForAnalyzeRepo,
		mockStockAPI,
		redisClient,
	)

	dailyPriceInteractor := usecase.NewStockBrandsDailyPriceInteractor(
		tx,
		stockBrandRepo,
		dailyPriceRepo,
		dailyPriceForAnalyzeRepo,
		mockStockAPI,
		redisClient,
		mockSlackAPI,
	)

	httpServer := driver.NewHTTPServer()
	stockBrandHandler := handler.NewStockBrandHandler(stockBrandInteractor, httpServer, zap.NewNop())
	stockPriceHandler := handler.NewStockPriceHandler(dailyPriceInteractor, httpServer, zap.NewNop())
	mux := router.NewRouter(stockPriceHandler, stockBrandHandler)
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
			name: "正常系: 全件取得",
			setup: func(t *testing.T) {
				brands := []*models.StockBrand{
					{
						ID:               "1",
						TickerSymbol:     "1301",
						Name:             "極洋",
						MarketCode:       "111",
						MarketName:       "プライム",
						Sector33Code:     "050",
						Sector33CodeName: "水産・農林業",
						Sector17Code:     "1",
						Sector17CodeName: "食品",
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					},
					{
						ID:               "2",
						TickerSymbol:     "1332",
						Name:             "日本水産",
						MarketCode:       "111",
						MarketName:       "プライム",
						Sector33Code:     "050",
						Sector33CodeName: "水産・農林業",
						Sector17Code:     "1",
						Sector17CodeName: "食品",
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					},
					{
						ID:               "3",
						TickerSymbol:     "1333",
						Name:             "マルハニチロ",
						MarketCode:       "121",
						MarketName:       "名古屋",
						Sector33Code:     "050",
						Sector33CodeName: "水産・農林業",
						Sector17Code:     "1",
						Sector17CodeName: "食品",
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					},
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				require.NoError(t, err)
			},
			query:      "",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res handler.GetStockBrandsResponse
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				require.Len(t, res.StockBrands, 3)
				assert.Equal(t, "1301", res.StockBrands[0].TickerSymbol)
				assert.Equal(t, "1332", res.StockBrands[1].TickerSymbol)
				assert.Equal(t, "1333", res.StockBrands[2].TickerSymbol)
				// ページネーションが指定されていないため、Paginationはnil
				assert.Nil(t, res.Pagination)
			},
		},
		{
			name: "正常系: ページネーション付き取得",
			setup: func(t *testing.T) {
				brands := []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1301",
						Name:         "極洋",
						MarketCode:   "111",
						MarketName:   "プライム",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "2",
						TickerSymbol: "1332",
						Name:         "日本水産",
						MarketCode:   "111",
						MarketName:   "プライム",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "3",
						TickerSymbol: "1333",
						Name:         "マルハニチロ",
						MarketCode:   "111",
						MarketName:   "プライム",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				require.NoError(t, err)
			},
			query:      "?symbol_from=1301&limit=2",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res handler.GetStockBrandsResponse
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				require.Len(t, res.StockBrands, 2)
				assert.Equal(t, "1332", res.StockBrands[0].TickerSymbol)
				assert.Equal(t, "1333", res.StockBrands[1].TickerSymbol)
				// ページネーション情報の確認
				require.NotNil(t, res.Pagination)
				assert.Equal(t, 2, res.Pagination.Limit)
				// 次のページがないのでnext_cursorはnil
				assert.Nil(t, res.Pagination.NextCursor)
			},
		},
		{
			name: "正常系: 主要市場のみフィルタリング",
			setup: func(t *testing.T) {
				brands := []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1301",
						Name:         "極洋",
						MarketCode:   "111", // プライム
						MarketName:   "プライム",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "2",
						TickerSymbol: "1332",
						Name:         "日本水産",
						MarketCode:   "112", // スタンダード
						MarketName:   "スタンダード",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "3",
						TickerSymbol: "1333",
						Name:         "マルハニチロ",
						MarketCode:   "121", // 他市場
						MarketName:   "名古屋",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "4",
						TickerSymbol: "1334",
						Name:         "テスト銘柄",
						MarketCode:   "113", // グロース
						MarketName:   "グロース",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				require.NoError(t, err)
			},
			query:      "?only_main_markets=true",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res handler.GetStockBrandsResponse
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				require.Len(t, res.StockBrands, 3)
				// 主要市場（111, 112, 113）のみが返されることを確認
				for _, brand := range res.StockBrands {
					assert.Contains(t, []string{"111", "112", "113"}, brand.MarketCode)
				}
				// ページネーションが指定されていないため、Paginationはnil
				assert.Nil(t, res.Pagination)
			},
		},
		{
			name: "正常系: ページネーション + 主要市場フィルタリング",
			setup: func(t *testing.T) {
				brands := []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1301",
						Name:         "極洋",
						MarketCode:   "111",
						MarketName:   "プライム",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "2",
						TickerSymbol: "1332",
						Name:         "日本水産",
						MarketCode:   "121",
						MarketName:   "名古屋",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "3",
						TickerSymbol: "1333",
						Name:         "マルハニチロ",
						MarketCode:   "112",
						MarketName:   "スタンダード",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				err := stockBrandRepo.UpsertStockBrands(context.Background(), brands)
				require.NoError(t, err)
			},
			query:      "?symbol_from=1301&limit=10&only_main_markets=true",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var res handler.GetStockBrandsResponse
				err := json.Unmarshal(body, &res)
				assert.NoError(t, err)
				// 1332は市場コード121なのでフィルタリングされる
				require.Len(t, res.StockBrands, 1)
				assert.Equal(t, "1333", res.StockBrands[0].TickerSymbol)
				assert.Equal(t, "112", res.StockBrands[0].MarketCode)
				// ページネーション情報の確認
				require.NotNil(t, res.Pagination)
				assert.Equal(t, 10, res.Pagination.Limit)
				// 次のカーソルはnil（最後のページ）
				assert.Nil(t, res.Pagination.NextCursor)
			},
		},
		{
			name:       "異常系: symbol_fromが長すぎる",
			setup:      func(t *testing.T) {},
			query:      "?symbol_from=12345678901",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "symbol_fromが長すぎます")
			},
		},
		{
			name:       "異常系: symbol_fromに不正な文字が含まれる",
			setup:      func(t *testing.T) {},
			query:      "?symbol_from=1234@",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "symbol_fromは英数字である必要があります")
			},
		},
		{
			name:       "異常系: limitが数値でない",
			setup:      func(t *testing.T) {},
			query:      "?limit=abc",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "limitは数値である必要があります")
			},
		},
		{
			name:       "異常系: limitが0以下",
			setup:      func(t *testing.T) {},
			query:      "?limit=0",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "limitは正の整数である必要があります")
			},
		},
		{
			name:       "異常系: limitが10000を超える",
			setup:      func(t *testing.T) {},
			query:      "?limit=10001",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "limitは10000以下である必要があります")
			},
		},
		{
			name:       "異常系: only_main_marketsがboolでない",
			setup:      func(t *testing.T) {},
			query:      "?only_main_markets=invalid",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "only_main_marketsはtrue/falseである必要があります")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テーブルクリア
			helper.TruncateAllTables(t, db)

			if tt.setup != nil {
				tt.setup(t)
			}

			// APIリクエスト実行
			resp, err := http.Get(ts.URL + "/stock-brands" + tt.query)
			require.NoError(t, err)
			defer resp.Body.Close()

			// ステータスコード確認
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			// レスポンスボディの読み取り
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(resp.Body)
			require.NoError(t, err)
			body := buf.Bytes()

			if tt.check != nil {
				tt.check(t, body)
			}
		})
	}
}

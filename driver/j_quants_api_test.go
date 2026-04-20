package driver

import (
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
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
)

func TestStockAPIClient_GetStockBrands(t *testing.T) {
	// Setup Redis (miniredis)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Backup original config
	originalJQuants := *config.GetJQuants()
	defer func() {
		*config.GetJQuants() = originalJQuants
	}()

	tests := []struct {
		name          string
		mockHandler   http.HandlerFunc
		want          []*gateway.StockBrand
		wantErr       bool
		mockSetup     func(mock *mock_driver.MockHTTPRequest)
	}{
		{
			name: "正常系: 銘柄一覧を取得できる",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/equities/master", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(http.StatusOK)
				resp := map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"Date":   "2023-01-01",
							"Code":   "13010",
							"CoName": "極洋",
							"Mkt":    "0111",
							"MktNm":  "プライム",
							"S17":    "1",
							"S17Nm":  "食品",
							"S33":    "0050",
							"S33Nm":  "水産・農林業",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			want: []*gateway.StockBrand{
				{
					Date:             time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
					Symbol:           "1301",
					CompanyName:      "極洋",
					MarketCode:       "111",
					MarketCodeName:   "プライム(国内株式)",
					Sector17Code:     "1",
					Sector17CodeName: "食品",
					Sector33Code:     "0050",
					Sector33CodeName: "水産・農林業",
				},
			},
			wantErr: false,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
		{
			name: "異常系: J-Quants APIがエラー(500)を返す",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    nil,
			wantErr: true,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
		{
			name: "異常系: レスポンスボディが不正なJSON",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			want:    nil,
			wantErr: true,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock Server
			ts := httptest.NewServer(tt.mockHandler)
			defer ts.Close()

			// Override Config
			config.GetJQuants().JQuantsBaseURLV2 = ts.URL
			config.GetJQuants().JQuantsBaseURLV2APIKey = "dummy-key"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockReq := mock_driver.NewMockHTTPRequest(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockReq)
			}

			c := NewStockAPIClient(mockReq, redisClient)

			got, err := c.GetStockBrands(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStockBrands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestStockAPIClient_GetAnnounceFinSchedule(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	originalJQuants := *config.GetJQuants()
	defer func() {
		*config.GetJQuants() = originalJQuants
	}()

	tests := []struct {
		name        string
		mockHandler http.HandlerFunc
		want        []*gateway.AnnounceFinScheduleResponseInfo
		wantErr     bool
		mockSetup   func(mock *mock_driver.MockHTTPRequest)
	}{
		{
			name: "正常系: 決算予定を取得できる",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/equities/earnings-calendar", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "dummy-key", r.Header.Get("x-api-key"))
				w.WriteHeader(http.StatusOK)
				resp := map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"Date":    "2023-01-04",
							"Code":    "13010",
							"CoName":  "極洋",
							"FY":      "2023",
							"SectorNm": "水産・農林業",
							"FQ":      "1Q",
							"Section": "Prime",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			want: []*gateway.AnnounceFinScheduleResponseInfo{
				{
					Date:          time.Date(2023, 1, 4, 0, 0, 0, 0, time.Local),
					Code:          "1301",
					CompanyName:   "極洋",
					FiscalYear:    "2023",
					SectorName:    "水産・農林業",
					FiscalQuarter: "1Q",
					Section:       "Prime",
				},
			},
			wantErr: false,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
		{
			name: "異常系: J-Quants APIがエラー(500)を返す",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    nil,
			wantErr: true,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.mockHandler)
			defer ts.Close()

			config.GetJQuants().JQuantsBaseURLV2 = ts.URL
			config.GetJQuants().JQuantsBaseURLV2APIKey = "dummy-key"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockReq := mock_driver.NewMockHTTPRequest(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockReq)
			}

			c := NewStockAPIClient(mockReq, redisClient)

			got, err := c.GetAnnounceFinSchedule(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAnnounceFinSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestStockAPIClient_getDailyPricesBySymbolAndRangeJQ(t *testing.T) {
	// Setup Redis (miniredis)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Backup original config
	originalJQuants := *config.GetJQuants()
	defer func() {
		*config.GetJQuants() = originalJQuants
	}()

	tests := []struct {
		name          string
		symbol        gateway.StockAPISymbol
		dateFrom      time.Time
		dateTo        time.Time
		mockHandler   http.HandlerFunc
		want          []*gateway.StockPrice
		wantErr       bool
		mockSetup     func(mock *mock_driver.MockHTTPRequest)
	}{
		{
			name:     "正常系: 日足データを取得できる",
			symbol:   gateway.StockAPISymbol("1301"),
			dateFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			dateTo:   time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/equities/bars/daily", r.URL.Path)
				assert.Equal(t, "1301", r.URL.Query().Get("code"))
				w.WriteHeader(http.StatusOK)
				resp := map[string]interface{}{
					"Data": []map[string]interface{}{
						{
							"Date":      "2023-01-04",
							"Code":      "13010",
							"O":         3000.0,
							"H":         3100.0,
							"L":         2900.0,
							"C":         3050.0,
							"Vo":        10000.0,
							"AdjH":      3100.0,
							"AdjL":      2900.0,
							"AdjO":      3000.0,
							"AdjC":      3050.0,
							"AdjVolume": 10000.0,
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			want: []*gateway.StockPrice{
				{
					Date:            time.Date(2023, 1, 4, 0, 0, 0, 0, time.Local),
					TickerSymbol:    "1301",
					Open:            decimal.NewFromFloat(3000.0),
					High:            decimal.NewFromFloat(3100.0),
					Low:             decimal.NewFromFloat(2900.0),
					Close:           decimal.NewFromFloat(3050.0),
					Volume:          10000,
					AdjustmentClose: decimal.NewFromFloat(3050.0),
				},
			},
			wantErr: false,
			mockSetup: func(mock *mock_driver.MockHTTPRequest) {
				mock.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock Server
			ts := httptest.NewServer(tt.mockHandler)
			defer ts.Close()

			// Override Config
			config.GetJQuants().JQuantsBaseURLV2 = ts.URL
			config.GetJQuants().JQuantsBaseURLV2APIKey = "dummy-key"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockReq := mock_driver.NewMockHTTPRequest(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockReq)
			}

			c := NewStockAPIClient(mockReq, redisClient)

			got, err := c.GetDailyPricesBySymbolAndRange(context.Background(), tt.symbol, tt.dateFrom, tt.dateTo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDailyPricesBySymbolAndRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if assert.Len(t, got, len(tt.want)) {
					for i, w := range tt.want {
						assert.Equal(t, w.TickerSymbol, got[i].TickerSymbol)
						assert.True(t, w.Open.Equal(got[i].Open), "Open price mismatch")
					}
				}
			}
		})
	}
}

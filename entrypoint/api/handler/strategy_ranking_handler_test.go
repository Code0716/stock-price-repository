package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestStrategyRankingHandler_GetStrategyRanking(t *testing.T) {
	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor
		httpServer func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer
	}
	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		wantStatusCode int
		wantBody       interface{}
	}{
		{
			name: "正常系: 計算済みランキングを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRanking(gomock.Any()).Return(&models.StrategyRanking{
						Computed:    true,
						ComputedAt:  "2026-06-01T00:00:00Z",
						Universe:    "main_markets",
						TotalStocks: 100,
						Params: models.BacktestParams{
							TakeProfit:  decimal.NewFromFloat(0.10),
							StopLoss:    decimal.NewFromFloat(0.05),
							MaxHoldDays: 20,
						},
						Items: []models.StrategyRankingItem{
							{
								Strategy:       "macd_bullish",
								Label:          "MACD強気",
								StockCount:     90,
								AvgTotalReturn: decimal.NewFromFloat(0.05),
							},
						},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.StrategyRanking{
				Computed:    true,
				ComputedAt:  "2026-06-01T00:00:00Z",
				Universe:    "main_markets",
				TotalStocks: 100,
				Params: models.BacktestParams{
					TakeProfit:  decimal.NewFromFloat(0.10),
					StopLoss:    decimal.NewFromFloat(0.05),
					MaxHoldDays: 20,
				},
				Items: []models.StrategyRankingItem{
					{
						Strategy:       "macd_bullish",
						Label:          "MACD強気",
						StockCount:     90,
						AvgTotalReturn: decimal.NewFromFloat(0.05),
					},
				},
			},
		},
		{
			name: "正常系: 未計算 (computed=false) を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRanking(gomock.Any()).Return(&models.StrategyRanking{
						Computed: false,
						Items:    []models.StrategyRankingItem{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.StrategyRanking{
				Computed: false,
				Items:    []models.StrategyRankingItem{},
			},
		},
		{
			name: "異常系: UseCaseがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRanking(gomock.Any()).Return(nil, errors.New("redis error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewStrategyRankingHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetStrategyRanking(w, tt.req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantStatusCode == http.StatusOK {
				wantJSON, err := json.Marshal(tt.wantBody)
				assert.NoError(t, err)
				assert.JSONEq(t, string(wantJSON), w.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestStrategyRankingHandler_GetStrategyRankingStocks(t *testing.T) {
	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor
		httpServer func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer
	}
	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		wantStatusCode int
		wantBody       interface{}
	}{
		{
			name: "正常系: 計算済みデータを limit 件で返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRankingStocks(gomock.Any(), gomock.Eq("macd_bullish"), gomock.Eq(10)).Return(&models.StrategyStocks{
						Computed:   true,
						ComputedAt: "2026-06-10T00:00:00Z",
						Strategy:   "macd_bullish",
						Label:      "MACD強気",
						TotalCount: 100,
						Items: []*models.StrategyStockResult{
							{TickerSymbol: "7203", Name: "トヨタ自動車", TotalReturn: decimal.NewFromFloat(0.15)},
						},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=macd_bullish&limit=10", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.StrategyStocks{
				Computed:   true,
				ComputedAt: "2026-06-10T00:00:00Z",
				Strategy:   "macd_bullish",
				Label:      "MACD強気",
				TotalCount: 100,
				Items: []*models.StrategyStockResult{
					{TickerSymbol: "7203", Name: "トヨタ自動車", TotalReturn: decimal.NewFromFloat(0.15)},
				},
			},
		},
		{
			name: "正常系: limit 省略時デフォルト100",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRankingStocks(gomock.Any(), gomock.Eq("bollinger_breakout"), gomock.Eq(100)).Return(&models.StrategyStocks{
						Computed: false,
						Strategy: "bollinger_breakout",
						Items:    []*models.StrategyStockResult{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=bollinger_breakout", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.StrategyStocks{
				Computed: false,
				Strategy: "bollinger_breakout",
				Items:    []*models.StrategyStockResult{},
			},
		},
		{
			name: "異常系: strategy パラメータなし → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					return mock_usecase.NewMockStrategyRankingInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "strategy パラメータは必須です\n",
		},
		{
			name: "異常系: 不正な strategy → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					return mock_usecase.NewMockStrategyRankingInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=invalid_strategy", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "strategy が不正です\n",
		},
		{
			name: "異常系: limit が整数でない → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					return mock_usecase.NewMockStrategyRankingInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=macd_bullish&limit=abc", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は整数で指定してください\n",
		},
		{
			name: "異常系: limit が範囲外（0） → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					return mock_usecase.NewMockStrategyRankingInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=macd_bullish&limit=0", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は 1 以上 2000 以下で指定してください\n",
		},
		{
			name: "異常系: limit が範囲外（2001） → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					return mock_usecase.NewMockStrategyRankingInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=macd_bullish&limit=2001", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は 1 以上 2000 以下で指定してください\n",
		},
		{
			name: "異常系: UseCaseがエラーを返す → 500",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStrategyRankingInteractor {
					m := mock_usecase.NewMockStrategyRankingInteractor(ctrl)
					m.EXPECT().GetStrategyRankingStocks(gomock.Any(), gomock.Eq("macd_bullish"), gomock.Eq(100)).Return(nil, errors.New("redis error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/strategy-ranking-stocks?strategy=macd_bullish", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewStrategyRankingHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetStrategyRankingStocks(w, tt.req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantStatusCode == http.StatusOK {
				wantJSON, err := json.Marshal(tt.wantBody)
				assert.NoError(t, err)
				assert.JSONEq(t, string(wantJSON), w.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

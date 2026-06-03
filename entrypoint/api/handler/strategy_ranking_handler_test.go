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

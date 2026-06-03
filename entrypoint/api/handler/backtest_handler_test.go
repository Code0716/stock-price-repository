package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestBacktestHandler_GetBacktest(t *testing.T) {
	date, _ := time.Parse(util.DateLayout, "2021-01-04")
	defaultParams := models.BacktestParams{
		TakeProfit:  decimal.NewFromFloat(0.10),
		StopLoss:    decimal.NewFromFloat(0.05),
		MaxHoldDays: 20,
	}

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor
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
			name: "正常系: 既定パラメータでバックテスト取得",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor {
					m := mock_usecase.NewMockBacktestInteractor(ctrl)
					m.EXPECT().
						GetBacktestComparison(gomock.Any(), "7203", &date, &date, defaultParams).
						Return(&models.BacktestComparison{
							Symbol:      "7203",
							From:        "2021-01-04",
							To:          "2021-01-04",
							TradingDays: 1,
							Params:      defaultParams,
							Strategies:  []models.StrategyBacktest{},
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "takeProfit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "stopLoss").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "maxHoldDays").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/backtest?symbol=7203&from=2021-01-04&to=2021-01-04", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.BacktestComparison{
				Symbol:      "7203",
				From:        "2021-01-04",
				To:          "2021-01-04",
				TradingDays: 1,
				Params:      defaultParams,
				Strategies:  []models.StrategyBacktest{},
			},
		},
		{
			name: "異常系: symbol未指定",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor {
					return mock_usecase.NewMockBacktestInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/backtest", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは必須です\n",
		},
		{
			name: "異常系: takeProfitが範囲外",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor {
					return mock_usecase.NewMockBacktestInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "takeProfit").Return("2")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/backtest?symbol=7203&takeProfit=2", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "takeProfitは0より大きく1以下である必要があります\n",
		},
		{
			name: "異常系: maxHoldDaysが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor {
					return mock_usecase.NewMockBacktestInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "takeProfit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "stopLoss").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "maxHoldDays").Return("0")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/backtest?symbol=7203&maxHoldDays=0", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "maxHoldDaysは1〜250である必要があります\n",
		},
		{
			name: "異常系: UseCaseエラー",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockBacktestInteractor {
					m := mock_usecase.NewMockBacktestInteractor(ctrl)
					m.EXPECT().
						GetBacktestComparison(gomock.Any(), "7203", &date, &date, defaultParams).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "takeProfit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "stopLoss").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "maxHoldDays").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/backtest?symbol=7203&from=2021-01-04&to=2021-01-04", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewBacktestHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())

			w := httptest.NewRecorder()
			h.GetBacktest(w, tt.req)

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

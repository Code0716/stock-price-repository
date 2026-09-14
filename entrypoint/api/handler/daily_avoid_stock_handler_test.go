package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

func TestDailyAvoidStockHandler_GetDailyAvoidStocks(t *testing.T) {
	asOfDate := time.Date(2026, 7, 24, 0, 0, 0, 0, time.Local)
	asOfDateStr := "2026-07-24"

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor
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
			name: "正常系: 指定日の避けるべき銘柄一覧を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					m := mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Eq(&asOfDate)).Return(&models.DailyAvoidStockDay{
						AsOfDate:            &asOfDateStr,
						RuleVersion:         "v1",
						UniverseSize:        100,
						ThresholdVolatility: decimal.RequireFromString("0.35"),
						Summary:             models.DailyAvoidStockSummary{FlaggedCount: 1, HighCount: 1},
						Items: []*models.DailyAvoidStockItem{
							{
								AvoidRank:     1,
								StockBrandID:  "b1",
								TickerSymbol:  "1000",
								Name:          "テスト銘柄",
								Severity:      models.DailyAvoidSeverityHigh,
								Reason:        models.DailyAvoidReasonHighVolatility,
								Volatility12M: decimal.RequireFromString("0.55"),
							},
						},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(&asOfDate, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks?date=2026-07-24", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.DailyAvoidStockDay{
				AsOfDate:            &asOfDateStr,
				RuleVersion:         "v1",
				UniverseSize:        100,
				ThresholdVolatility: decimal.RequireFromString("0.35"),
				Summary:             models.DailyAvoidStockSummary{FlaggedCount: 1, HighCount: 1},
				Items: []*models.DailyAvoidStockItem{
					{
						AvoidRank:     1,
						StockBrandID:  "b1",
						TickerSymbol:  "1000",
						Name:          "テスト銘柄",
						Severity:      models.DailyAvoidSeverityHigh,
						Reason:        models.DailyAvoidReasonHighVolatility,
						Volatility12M: decimal.RequireFromString("0.55"),
					},
				},
			},
		},
		{
			name: "正常系: date省略時もusecaseにnilを渡して200を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					m := mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Nil()).Return(&models.DailyAvoidStockDay{
						RuleVersion: "v1",
						Items:       []*models.DailyAvoidStockItem{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.DailyAvoidStockDay{
				RuleVersion: "v1",
				Items:       []*models.DailyAvoidStockItem{},
			},
		},
		{
			name: "異常系: dateの形式が不正なら400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					return mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, errors.New("parse error"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks?date=bad", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "dateの日付形式が不正です (YYYY-MM-DD)\n",
		},
		{
			name: "異常系: usecaseエラーは500",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					m := mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Nil()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewDailyAvoidStockHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetDailyAvoidStocks(w, tt.req)

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

func TestDailyAvoidStockHandler_GetDailyAvoidStockDates(t *testing.T) {
	type fields struct {
		usecase func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor
	}
	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		wantStatusCode int
		wantBody       interface{}
	}{
		{
			name: "正常系: limit省略時は既定値90で呼ぶ",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					m := mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
					m.EXPECT().GetAsOfDates(gomock.Any(), gomock.Eq(dailyAvoidStockDatesDefaultLimit)).
						Return(&models.DailyAvoidStockDates{Dates: []string{"2026-07-24"}}, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks/dates", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       &models.DailyAvoidStockDates{Dates: []string{"2026-07-24"}},
		},
		{
			name: "正常系: limit指定",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					m := mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
					m.EXPECT().GetAsOfDates(gomock.Any(), gomock.Eq(30)).
						Return(&models.DailyAvoidStockDates{Dates: []string{}}, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks/dates?limit=30", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       &models.DailyAvoidStockDates{Dates: []string{}},
		},
		{
			name: "異常系: limitが整数でないと400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					return mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks/dates?limit=abc", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は整数で指定してください\n",
		},
		{
			name: "異常系: limitが上限超過だと400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyAvoidStockInteractor {
					return mock_usecase.NewMockDailyAvoidStockInteractor(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-avoid-stocks/dates?limit=401", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は 1 以上 400 以下で指定してください\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewDailyAvoidStockHandler(tt.fields.usecase(ctrl), mock_driver.NewMockHTTPServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetDailyAvoidStockDates(w, tt.req)

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

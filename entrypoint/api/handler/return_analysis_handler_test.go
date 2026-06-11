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

func TestReturnAnalysisHandler_GetReturnAnalysis(t *testing.T) {
	date, _ := time.Parse(util.DateLayout, "2024-01-04")

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor
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
			name: "正常系: benchmark省略→nikkei",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					m := mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
					m.EXPECT().
						GetReturnAnalysis(gomock.Any(), "7203", &date, &date, models.BenchmarkNikkei).
						Return(&models.ReturnAnalysis{
							Symbol:           "7203",
							Benchmark:        models.BenchmarkNikkei,
							From:             "2024-01-04",
							To:               "2024-01-04",
							TradingDays:      1,
							CumulativeReturn: decimal.NewFromFloat(0.21),
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "benchmark").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&from=2024-01-04&to=2024-01-04", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.ReturnAnalysis{
				Symbol:           "7203",
				Benchmark:        models.BenchmarkNikkei,
				From:             "2024-01-04",
				To:               "2024-01-04",
				TradingDays:      1,
				CumulativeReturn: decimal.NewFromFloat(0.21),
			},
		},
		{
			name: "正常系: benchmark=topix",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					m := mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
					m.EXPECT().
						GetReturnAnalysis(gomock.Any(), "7203", &date, &date, models.BenchmarkTopix).
						Return(&models.ReturnAnalysis{
							Symbol:      "7203",
							Benchmark:   models.BenchmarkTopix,
							From:        "2024-01-04",
							To:          "2024-01-04",
							TradingDays: 1,
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "benchmark").Return(models.BenchmarkTopix)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&benchmark=topix", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.ReturnAnalysis{
				Symbol:      "7203",
				Benchmark:   models.BenchmarkTopix,
				From:        "2024-01-04",
				To:          "2024-01-04",
				TradingDays: 1,
			},
		},
		{
			name: "異常系: benchmark不正値→400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "benchmark").Return("sp500")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&benchmark=sp500", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "benchmarkは\"nikkei\"または\"topix\"である必要があります\n",
		},
		{
			name: "異常系: symbolが指定されていない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは必須です\n",
		},
		{
			name: "異常系: symbolが長すぎる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("12345678901")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=12345678901", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルが長すぎます\n",
		},
		{
			name: "異常系: symbolに不正な文字",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203@")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203@", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは英数字である必要があります\n",
		},
		{
			name: "異常系: fromの日付フォーマットが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&from=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromの日付形式が不正です\n",
		},
		{
			name: "異常系: toの日付フォーマットが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					return mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&from=2024-01-04&to=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "toの日付形式が不正です\n",
		},
		{
			name: "異常系: UseCaseがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockReturnAnalysisInteractor {
					m := mock_usecase.NewMockReturnAnalysisInteractor(ctrl)
					m.EXPECT().
						GetReturnAnalysis(gomock.Any(), "7203", &date, &date, models.BenchmarkNikkei).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "benchmark").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/return-analysis?symbol=7203&from=2024-01-04&to=2024-01-04", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewReturnAnalysisHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())

			w := httptest.NewRecorder()
			h.GetReturnAnalysis(w, tt.req)

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

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

func TestSignalPerformanceHandler_GetSignalPerformance(t *testing.T) {
	// 固定日付
	fixedDate, _ := time.Parse(util.DateLayout, "2024-03-31")

	// 正常系レスポンスのサンプル
	okResult := &models.SignalPerformance{
		From:      fixedDate.AddDate(0, 0, -90),
		To:        fixedDate,
		Horizons:  []int{5, 10, 20},
		Summaries: []*models.SignalPerformanceSummary{},
		Signals:   []*models.EvaluatedSignal{},
	}

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor
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
			name: "正常系: to/from 指定 → usecase に渡る",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					m := mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
					from := fixedDate.AddDate(0, 0, -90)
					m.EXPECT().GetSignalPerformance(gomock.Any(), gomock.Eq(&models.SignalPerformanceFilter{
						From: from,
						To:   fixedDate,
					})).Return(okResult, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "method").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "action").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?to=2024-03-31", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       okResult,
		},
		{
			name: "正常系: パラメータ省略時 to=今日 / from=to-90日 が usecase に渡る",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					m := mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
					// 実際の handler が計算する today/from と同じ値で比較
					now := time.Now()
					today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
					from := today.AddDate(0, 0, -90)
					m.EXPECT().GetSignalPerformance(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ interface{}, f *models.SignalPerformanceFilter) (*models.SignalPerformance, error) {
							// to は今日
							assert.Equal(t, today.Truncate(24*time.Hour), f.To.Truncate(24*time.Hour))
							// from は to-90日
							assert.Equal(t, from.Truncate(24*time.Hour), f.From.Truncate(24*time.Hour))
							return &models.SignalPerformance{
								From:      f.From,
								To:        f.To,
								Horizons:  []int{5, 10, 20},
								Summaries: []*models.SignalPerformanceSummary{},
								Signals:   []*models.EvaluatedSignal{},
							}, nil
						},
					)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "method").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "action").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance", nil),
			wantStatusCode: http.StatusOK,
		},
		{
			name: "正常系: method 指定時 → usecase にフィルタが渡る",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					m := mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
					from := fixedDate.AddDate(0, 0, -90)
					m.EXPECT().GetSignalPerformance(gomock.Any(), gomock.Eq(&models.SignalPerformanceFilter{
						From:   from,
						To:     fixedDate,
						Method: "find_macd_bullish_stock_v1",
					})).Return(&models.SignalPerformance{
						From:     from,
						To:       fixedDate,
						Horizons: []int{5, 10, 20},
						Summaries: []*models.SignalPerformanceSummary{
							{
								Method:      "find_macd_bullish_stock_v1",
								SignalCount: 3,
								Stats: map[int]*models.HorizonStats{
									5:  {EvaluatedCount: 3, WinRate: decimal.NewFromFloat(0.667)},
									10: {EvaluatedCount: 3, WinRate: decimal.NewFromFloat(0.333)},
									20: {EvaluatedCount: 3, WinRate: decimal.NewFromFloat(0.667)},
								},
							},
						},
						Signals: []*models.EvaluatedSignal{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "method").Return("find_macd_bullish_stock_v1")
					m.EXPECT().GetQueryParam(gomock.Any(), "action").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?to=2024-03-31&method=find_macd_bullish_stock_v1", nil),
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: from > to → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					return mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					from := fixedDate.AddDate(0, 0, 10) // to より後
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&from, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?from=2024-04-10&to=2024-03-31", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromはtoより前の日付を指定してください\n",
		},
		{
			name: "異常系: 期間 366 日超 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					return mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					from := fixedDate.AddDate(-1, 0, -2) // 367日以上前
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&from, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?from=2023-01-01&to=2024-03-31", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "期間は最大366日以内で指定してください\n",
		},
		{
			name: "異常系: to の日付形式が不正 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					return mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?to=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "toの日付形式が不正です（YYYY-MM-DD）\n",
		},
		{
			name: "異常系: from の日付形式が不正 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					return mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance?from=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromの日付形式が不正です（YYYY-MM-DD）\n",
		},
		{
			name: "異常系: usecase がエラーを返す → 500",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSignalPerformanceInteractor {
					m := mock_usecase.NewMockSignalPerformanceInteractor(ctrl)
					m.EXPECT().GetSignalPerformance(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&fixedDate, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, nil)
					m.EXPECT().GetQueryParam(gomock.Any(), "method").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "action").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/signal-performance", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewSignalPerformanceHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetSignalPerformance(w, tt.req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantBody == nil {
				return
			}
			if tt.wantStatusCode == http.StatusOK {
				if tt.wantBody != nil {
					wantJSON, err := json.Marshal(tt.wantBody)
					assert.NoError(t, err)
					assert.JSONEq(t, string(wantJSON), w.Body.String())
				}
			} else {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

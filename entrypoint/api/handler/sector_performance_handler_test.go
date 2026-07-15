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

func TestSectorPerformanceHandler_GetSectorPerformance(t *testing.T) {
	fixedTo, _ := time.ParseInLocation(util.DateLayout, "2024-03-31", time.Local)
	fixedFrom := fixedTo.AddDate(0, 0, -90)

	pr := decimal.NewFromFloat(0.05)
	okResult33 := &models.SectorPerformance{
		Granularity: "33",
		From:        "2024-01-01",
		To:          "2024-03-31",
		Sectors: []*models.SectorPerformanceItem{
			{
				SectorCode:   "3700",
				SectorName:   "輸送用機器",
				PeriodReturn: &pr,
				LatestClose:  &pr,
				LatestDate:   "2024-03-31",
			},
		},
	}
	okResult17 := &models.SectorPerformance{
		Granularity: "17",
		From:        "2024-01-01",
		To:          "2024-03-31",
		Sectors: []*models.SectorPerformanceItem{
			{
				SectorCode:   "6",
				SectorName:   "自動車・輸送機",
				PeriodReturn: &pr,
				LatestClose:  &pr,
				LatestDate:   "2024-03-31",
			},
		},
	}

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor
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
			name: "正常系: granularity=33 / to/from 指定 → usecase に渡る",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					m := mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
					m.EXPECT().GetSectorPerformance(gomock.Any(), gomock.Eq(fixedFrom), gomock.Eq(fixedTo), gomock.Eq("33")).Return(okResult33, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=2024-01-01&to=2024-03-31&granularity=33", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       okResult33,
		},
		{
			name: "正常系: granularity=17 → usecase に 17 が渡る",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					m := mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
					m.EXPECT().GetSectorPerformance(gomock.Any(), gomock.Eq(fixedFrom), gomock.Eq(fixedTo), gomock.Eq("17")).Return(okResult17, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=2024-01-01&to=2024-03-31&granularity=17", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       okResult17,
		},
		{
			name: "正常系: granularity 省略時 → 33 がデフォルト",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					m := mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
					m.EXPECT().GetSectorPerformance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq("33")).Return(&models.SectorPerformance{
						Granularity: "33",
						From:        fixedFrom.Format(util.DateLayout),
						To:          fixedTo.Format(util.DateLayout),
						Sectors:     []*models.SectorPerformanceItem{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=2024-01-01&to=2024-03-31", nil),
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: granularity が不正値 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					return mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?granularity=99", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "granularityは\"33\"または\"17\"を指定してください\n",
		},
		{
			name: "正常系: パラメータ省略時 to=今日 / from=to-90日",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					m := mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
					m.EXPECT().GetSectorPerformance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq("33")).DoAndReturn(
						func(_ interface{}, from, to time.Time, granularity string) (*models.SectorPerformance, error) {
							now := time.Now()
							today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
							expectedFrom := today.AddDate(0, 0, -90)
							assert.Equal(t, today.Truncate(24*time.Hour), to.Truncate(24*time.Hour))
							assert.Equal(t, expectedFrom.Truncate(24*time.Hour), from.Truncate(24*time.Hour))
							return &models.SectorPerformance{
								Granularity: "33",
								From:        from.Format(util.DateLayout),
								To:          to.Format(util.DateLayout),
								Sectors:     []*models.SectorPerformanceItem{},
							}, nil
						},
					)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance", nil),
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: from > to → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					return mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=2024-04-10&to=2024-03-31", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromはto以前の日付である必要があります\n",
		},
		{
			name: "異常系: 期間 366 日超 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					return mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return mock_driver.NewMockHTTPServer(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=2023-01-01&to=2024-03-31", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "期間は最大366日以内で指定してください\n",
		},
		{
			name: "異常系: to の日付形式が不正 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					return mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?to=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "toの日付形式が不正です (YYYY-MM-DD)\n",
		},
		{
			name: "異常系: from の日付形式が不正 → 400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					return mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance?from=invalid", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromの日付形式が不正です (YYYY-MM-DD)\n",
		},
		{
			name: "異常系: usecase がエラーを返す → 500",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockSectorPerformanceInteractor {
					m := mock_usecase.NewMockSectorPerformanceInteractor(ctrl)
					m.EXPECT().GetSectorPerformance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/sector-performance", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewSectorPerformanceHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetSectorPerformance(w, tt.req)

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

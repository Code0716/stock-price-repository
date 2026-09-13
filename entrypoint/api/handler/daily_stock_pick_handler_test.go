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

func TestDailyStockPickHandler_GetDailyStockPicks(t *testing.T) {
	pickDate := time.Date(2026, 7, 24, 0, 0, 0, 0, time.Local)
	pickDateStr := "2026-07-24"

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor
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
			name: "正常系: 指定日の推奨一覧を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					m := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Eq(&pickDate)).Return(&models.DailyStockPickDay{
						PickDate:     &pickDateStr,
						ScoreVersion: "v1",
						Evaluated:    false,
						Summary:      models.DailyStockPickStatSummary{Total: 1, PendingCount: 1},
						Items: []*models.DailyStockPickItem{
							{
								PickRank:     1,
								StockBrandID: "b1",
								TickerSymbol: "1000",
								Name:         "テスト銘柄",
								Score:        decimal.RequireFromString("82.50"),
								ScoreVersion: "v1",
								SignalCount:  1,
								Strategies: []*models.DailyStockPickStrategy{
									{Key: "macd_bullish", Label: "MACD強気"},
								},
							},
						},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(&pickDate, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks?date=2026-07-24", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.DailyStockPickDay{
				PickDate:     &pickDateStr,
				ScoreVersion: "v1",
				Evaluated:    false,
				Summary:      models.DailyStockPickStatSummary{Total: 1, PendingCount: 1},
				Items: []*models.DailyStockPickItem{
					{
						PickRank:     1,
						StockBrandID: "b1",
						TickerSymbol: "1000",
						Name:         "テスト銘柄",
						Score:        decimal.RequireFromString("82.50"),
						ScoreVersion: "v1",
						SignalCount:  1,
						Strategies: []*models.DailyStockPickStrategy{
							{Key: "macd_bullish", Label: "MACD強気"},
						},
					},
				},
			},
		},
		{
			name: "正常系: date省略時もusecaseにnilを渡して200を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					m := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Nil()).Return(&models.DailyStockPickDay{
						ScoreVersion: "v1",
						Items:        []*models.DailyStockPickItem{},
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.DailyStockPickDay{
				ScoreVersion: "v1",
				Items:        []*models.DailyStockPickItem{},
			},
		},
		{
			name: "異常系: dateの形式が不正なら400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					return mock_usecase.NewMockDailyStockPickInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, errors.New("parse error"))
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks?date=bad", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "dateの日付形式が不正です (YYYY-MM-DD)\n",
		},
		{
			name: "異常系: usecaseエラーは500",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					m := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
					m.EXPECT().GetDay(gomock.Any(), gomock.Nil()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "date", util.DateLayout).Return(nil, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewDailyStockPickHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetDailyStockPicks(w, tt.req)

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

func TestDailyStockPickHandler_GetDailyStockPickDates(t *testing.T) {
	type fields struct {
		usecase func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor
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
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					m := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
					m.EXPECT().GetPickDates(gomock.Any(), gomock.Eq(dailyStockPickDatesDefaultLimit)).
						Return(&models.DailyStockPickDates{Dates: []string{"2026-07-24"}}, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks/dates", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       &models.DailyStockPickDates{Dates: []string{"2026-07-24"}},
		},
		{
			name: "正常系: limit指定",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					m := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
					m.EXPECT().GetPickDates(gomock.Any(), gomock.Eq(30)).
						Return(&models.DailyStockPickDates{Dates: []string{}}, nil)
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks/dates?limit=30", nil),
			wantStatusCode: http.StatusOK,
			wantBody:       &models.DailyStockPickDates{Dates: []string{}},
		},
		{
			name: "異常系: limitが整数でないと400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					return mock_usecase.NewMockDailyStockPickInteractor(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks/dates?limit=abc", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は整数で指定してください\n",
		},
		{
			name: "異常系: limitが上限超過だと400",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockDailyStockPickInteractor {
					return mock_usecase.NewMockDailyStockPickInteractor(ctrl)
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/daily-stock-picks/dates?limit=401", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limit は 1 以上 400 以下で指定してください\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewDailyStockPickHandler(tt.fields.usecase(ctrl), mock_driver.NewMockHTTPServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetDailyStockPickDates(w, tt.req)

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

func TestDailyStockPickHandler_GetDailyStockPickStats(t *testing.T) {
	t.Run("正常系: score_version省略時は空文字でusecaseに渡す（usecase側で既定値を解決する）", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		u := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
		u.EXPECT().GetStats(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Eq("")).
			Return(&models.DailyStockPickStats{
				ScoreVersion: "v1",
				Daily:        []*models.DailyStockPickDailyStat{},
				ScoreBands:   []*models.DailyStockPickScoreBand{},
			}, nil)

		s := mock_driver.NewMockHTTPServer(ctrl)
		s.EXPECT().GetQueryParam(gomock.Any(), "score_version").Return("")

		h := NewDailyStockPickHandler(u, s, zap.NewNop())
		w := httptest.NewRecorder()
		h.GetDailyStockPickStats(w, httptest.NewRequest(http.MethodGet, "/daily-stock-picks/stats", nil))

		assert.Equal(t, http.StatusOK, w.Code)
		wantJSON, err := json.Marshal(&models.DailyStockPickStats{
			ScoreVersion: "v1",
			Daily:        []*models.DailyStockPickDailyStat{},
			ScoreBands:   []*models.DailyStockPickScoreBand{},
		})
		assert.NoError(t, err)
		assert.JSONEq(t, string(wantJSON), w.Body.String())
	})

	t.Run("正常系: score_version指定がそのまま渡る", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		u := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
		u.EXPECT().GetStats(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Eq("v2")).
			Return(&models.DailyStockPickStats{ScoreVersion: "v2"}, nil)

		s := mock_driver.NewMockHTTPServer(ctrl)
		s.EXPECT().GetQueryParam(gomock.Any(), "score_version").Return("v2")

		h := NewDailyStockPickHandler(u, s, zap.NewNop())
		w := httptest.NewRecorder()
		h.GetDailyStockPickStats(w, httptest.NewRequest(http.MethodGet, "/daily-stock-picks/stats?score_version=v2", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系: from > to は400", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		u := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
		s := mock_driver.NewMockHTTPServer(ctrl)

		h := NewDailyStockPickHandler(u, s, zap.NewNop())
		w := httptest.NewRecorder()
		h.GetDailyStockPickStats(w, httptest.NewRequest(http.MethodGet, "/daily-stock-picks/stats?from=2026-07-24&to=2026-07-01", nil))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系: usecaseエラーは500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		u := mock_usecase.NewMockDailyStockPickInteractor(ctrl)
		u.EXPECT().GetStats(gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Any()).Return(nil, errors.New("db error"))

		s := mock_driver.NewMockHTTPServer(ctrl)
		s.EXPECT().GetQueryParam(gomock.Any(), "score_version").Return("")

		h := NewDailyStockPickHandler(u, s, zap.NewNop())
		w := httptest.NewRecorder()
		h.GetDailyStockPickStats(w, httptest.NewRequest(http.MethodGet, "/daily-stock-picks/stats", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "内部サーバーエラー\n", w.Body.String())
	})
}

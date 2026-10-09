package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestMarketCalendarHandler_GetMarketCalendar(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name           string
		method         string
		target         string
		usecase        func(ctrl *gomock.Controller) *mock_usecase.MockMarketCalendarInteractor
		wantStatusCode int
		wantBody       string
	}{
		{
			name:   "正常系: 営業日とイベントを返す",
			method: http.MethodGet,
			target: "/market/calendar?from=2026-10-01&to=2026-10-31",
			usecase: func(ctrl *gomock.Controller) *mock_usecase.MockMarketCalendarInteractor {
				m := mock_usecase.NewMockMarketCalendarInteractor(ctrl)
				m.EXPECT().GetMarketCalendar(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(&usecase.MarketCalendar{
					Days: []*models.MarketCalendarDay{
						{Date: time.Date(2026, 10, 9, 0, 0, 0, 0, time.Local), HolDiv: 1},
						{Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local), HolDiv: 0},
					},
					Events: []*models.MarketEvent{
						{Date: time.Date(2026, 10, 9, 0, 0, 0, 0, time.Local), Kind: "sq_mini", Label: "SQ", Source: "jquants_calc"},
					},
				}, nil)
				return m
			},
			wantStatusCode: http.StatusOK,
			wantBody:       `{"days":[{"date":"2026-10-09","holDiv":1},{"date":"2026-10-10","holDiv":0}],"events":[{"date":"2026-10-09","kind":"sq_mini","label":"SQ"}]}`,
		},
		{
			name:   "正常系: データが無ければ空配列（null にしない）",
			method: http.MethodGet,
			target: "/market/calendar?from=2026-10-01&to=2026-10-31",
			usecase: func(ctrl *gomock.Controller) *mock_usecase.MockMarketCalendarInteractor {
				m := mock_usecase.NewMockMarketCalendarInteractor(ctrl)
				m.EXPECT().GetMarketCalendar(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(&usecase.MarketCalendar{}, nil)
				return m
			},
			wantStatusCode: http.StatusOK,
			wantBody:       `{"days":[],"events":[]}`,
		},
		{
			name:           "異常系: GET 以外は 405",
			method:         http.MethodPost,
			target:         "/market/calendar?from=2026-10-01&to=2026-10-31",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "異常系: from が無い",
			method:         http.MethodGet,
			target:         "/market/calendar?to=2026-10-31",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: to が無い",
			method:         http.MethodGet,
			target:         "/market/calendar?from=2026-10-01",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: 日付形式が不正",
			method:         http.MethodGet,
			target:         "/market/calendar?from=2026/10/01&to=2026-10-31",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: from が to より後",
			method:         http.MethodGet,
			target:         "/market/calendar?from=2026-11-01&to=2026-10-31",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "異常系: 範囲が400日を超える",
			method:         http.MethodGet,
			target:         "/market/calendar?from=2025-01-01&to=2026-12-31",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "異常系: usecase がエラー",
			method: http.MethodGet,
			target: "/market/calendar?from=2026-10-01&to=2026-10-31",
			usecase: func(ctrl *gomock.Controller) *mock_usecase.MockMarketCalendarInteractor {
				m := mock_usecase.NewMockMarketCalendarInteractor(ctrl)
				m.EXPECT().GetMarketCalendar(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db down"))
				return m
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var u usecase.MarketCalendarInteractor = mock_usecase.NewMockMarketCalendarInteractor(ctrl)
			if tt.usecase != nil {
				u = tt.usecase(ctrl)
			}
			h := NewMarketCalendarHandler(u, zap.NewNop())

			w := httptest.NewRecorder()
			h.GetMarketCalendar(w, httptest.NewRequest(tt.method, tt.target, nil))

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

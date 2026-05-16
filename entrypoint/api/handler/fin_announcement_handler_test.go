package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
)

func setupFinAnnouncementMockHTTPServer(ctrl *gomock.Controller, params map[string]string) *mock_driver.MockHTTPServer {
	m := mock_driver.NewMockHTTPServer(ctrl)
	keys := []string{"symbol", "from", "to", "page", "limit"}
	for _, k := range keys {
		val := params[k]
		m.EXPECT().GetQueryParam(gomock.Any(), k).Return(val).AnyTimes()
	}
	return m
}

func TestFinAnnouncementHandler_GetFinAnnouncements(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	sample := &models.FinAnnouncement{
		ID:               "id-1",
		TickerSymbol:     "7203",
		AnnouncementDate: now,
		FiscalYear:       "FY2025",
		FiscalQuarter:    "2Q",
	}

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor
		httpServer func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name: "正常系: デフォルトパラメータ",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().GetFinAnnouncements(gomock.Any(), gomock.Any()).Return(&models.PaginatedFinAnnouncements{
						Announcements: []*models.FinAnnouncement{sample},
						Page:          1,
						Limit:         100,
						Total:         1,
						TotalPages:    1,
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupFinAnnouncementMockHTTPServer(ctrl, map[string]string{})
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "異常系: 不正な from 日付",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupFinAnnouncementMockHTTPServer(ctrl, map[string]string{
						"from": "not-a-date",
					})
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "異常系: usecase エラー",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().GetFinAnnouncements(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupFinAnnouncementMockHTTPServer(ctrl, map[string]string{})
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := &FinAnnouncementHandler{
				usecase:    tt.fields.usecase(ctrl),
				httpServer: tt.fields.httpServer(ctrl),
				logger:     zap.NewNop(),
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/fin-announcements", nil)
			h.GetFinAnnouncements(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var body GetFinAnnouncementsResponse
				assert.NoError(t, json.NewDecoder(w.Body).Decode(&body))
				assert.NotNil(t, body.Pagination)
			}
		})
	}
}

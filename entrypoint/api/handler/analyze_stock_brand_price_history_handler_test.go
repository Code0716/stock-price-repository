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
)

func setupAnalyzeHistoryMockHTTPServer(ctrl *gomock.Controller, params map[string]string) *mock_driver.MockHTTPServer {
	m := mock_driver.NewMockHTTPServer(ctrl)
	keys := []string{"symbol", "action", "method", "sort_by", "order", "page", "limit"}
	for _, k := range keys {
		val := params[k]
		m.EXPECT().GetQueryParam(gomock.Any(), k).Return(val).AnyTimes()
	}
	return m
}

func TestAnalyzeStockBrandPriceHistoryHandler_GetAnalyzeStockBrandPriceHistories(t *testing.T) {
	now := time.Now()
	sampleHistory := &models.AnalyzeStockBrandPriceHistory{
		ID:           "id-1",
		TickerSymbol: "1234",
		TradePrice:   decimal.NewFromFloat(1000),
		CurrentPrice: decimal.NewFromFloat(1200),
		Action:       "Buy",
		Method:       "find_macd_bullish_stock_v1",
		CreatedAt:    now,
	}

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor
		httpServer func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
		wantPagination *AnalyzeHistoryPaginationInfo
	}{
		{
			name: "正常系: デフォルトパラメータ",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().GetAnalyzeStockBrandPriceHistories(gomock.Any(), gomock.Eq(&models.AnalyzeStockBrandPriceHistoryFilter{
						SortBy: models.AnalyzeStockBrandPriceHistorySortByCreatedAt,
						Order:  models.AnalyzeStockBrandPriceHistoryOrderDesc,
						Page:   1,
						Limit:  100,
					})).Return(&models.PaginatedAnalyzeStockBrandPriceHistories{
						Histories:  []*models.AnalyzeStockBrandPriceHistory{sampleHistory},
						Page:       1,
						Limit:      100,
						Total:      1,
						TotalPages: 1,
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{})
				},
			},
			wantStatusCode: http.StatusOK,
			wantPagination: &AnalyzeHistoryPaginationInfo{Page: 1, Limit: 100, Total: 1, TotalPages: 1},
		},
		{
			name: "正常系: sort_by=profit&order=asc&page=2&limit=20",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().GetAnalyzeStockBrandPriceHistories(gomock.Any(), gomock.Eq(&models.AnalyzeStockBrandPriceHistoryFilter{
						SortBy: models.AnalyzeStockBrandPriceHistorySortByProfit,
						Order:  models.AnalyzeStockBrandPriceHistoryOrderAsc,
						Page:   2,
						Limit:  20,
					})).Return(&models.PaginatedAnalyzeStockBrandPriceHistories{
						Histories:  []*models.AnalyzeStockBrandPriceHistory{},
						Page:       2,
						Limit:      20,
						Total:      25,
						TotalPages: 2,
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{
						"sort_by": "profit",
						"order":   "asc",
						"page":    "2",
						"limit":   "20",
					})
				},
			},
			wantStatusCode: http.StatusOK,
			wantPagination: &AnalyzeHistoryPaginationInfo{Page: 2, Limit: 20, Total: 25, TotalPages: 2},
		},
		{
			name: "異常系: 不正な sort_by",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{
						"sort_by": "invalid_key",
					})
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "異常系: 不正な order",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{
						"order": "invalid_order",
					})
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "異常系: page が 0",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{
						"page": "0",
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
					m.EXPECT().GetAnalyzeStockBrandPriceHistories(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					return setupAnalyzeHistoryMockHTTPServer(ctrl, map[string]string{})
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := &AnalyzeStockBrandPriceHistoryHandler{
				usecase:    tt.fields.usecase(ctrl),
				httpServer: tt.fields.httpServer(ctrl),
				logger:     zap.NewNop(),
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/analyze-stock-brand-price-histories", nil)
			h.GetAnalyzeStockBrandPriceHistories(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantPagination != nil {
				var body GetAnalyzeStockBrandPriceHistoriesResponse
				require := assert.New(t)
				require.NoError(json.NewDecoder(w.Body).Decode(&body))
				assert.Equal(t, tt.wantPagination, body.Pagination)
			}
		})
	}
}

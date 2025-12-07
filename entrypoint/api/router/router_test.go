package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestNewRouter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDailyPriceUsecase := mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
	mockStockBrandUsecase := mock_usecase.NewMockStockBrandInteractor(ctrl)
	mockHTTPServer := mock_driver.NewMockHTTPServer(ctrl)

	// ハンドラーが正しく呼び出されるかを確認するため、
	// ハンドラー内部で最初に呼ばれる GetQueryParam が実行されることを期待する。
	mockHTTPServer.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("").Times(1)

	stockPriceHandler := handler.NewStockPriceHandler(mockDailyPriceUsecase, mockHTTPServer, zap.NewNop())
	stockBrandHandler := handler.NewStockBrandHandler(mockStockBrandUsecase, mockHTTPServer, zap.NewNop())
	mux := NewRouter(stockPriceHandler, stockBrandHandler)

	req := httptest.NewRequest(http.MethodGet, "/daily-prices", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// ハンドラーが実行された結果、symbolがないので400が返るはず
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewRouter_WithNilStockBrandHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDailyPriceUsecase := mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
	mockHTTPServer := mock_driver.NewMockHTTPServer(ctrl)

	stockPriceHandler := handler.NewStockPriceHandler(mockDailyPriceUsecase, mockHTTPServer, zap.NewNop())
	mux := NewRouter(stockPriceHandler, nil)

	// /stock-brands エンドポイントにアクセスしても、404が返るはず（パニックしない）
	req := httptest.NewRequest(http.MethodGet, "/stock-brands", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_WithNilStockPriceHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStockBrandUsecase := mock_usecase.NewMockStockBrandInteractor(ctrl)
	mockHTTPServer := mock_driver.NewMockHTTPServer(ctrl)

	stockBrandHandler := handler.NewStockBrandHandler(mockStockBrandUsecase, mockHTTPServer, zap.NewNop())
	mux := NewRouter(nil, stockBrandHandler)

	// /daily-prices エンドポイントにアクセスしても、404が返るはず（パニックしない）
	req := httptest.NewRequest(http.MethodGet, "/daily-prices", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_WithBothNil(t *testing.T) {
	mux := NewRouter(nil, nil)

	// どちらのエンドポイントにアクセスしても、404が返るはず（パニックしない）
	tests := []struct {
		name string
		path string
	}{
		{"daily-prices endpoint", "/daily-prices"},
		{"stock-brands endpoint", "/stock-brands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

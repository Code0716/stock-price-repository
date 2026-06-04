package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestValuationHandler_GetValuation(t *testing.T) {
	per := decimal.NewFromFloat(10)
	pbr := decimal.NewFromFloat(2)
	roe := decimal.NewFromFloat(0.2)
	fwdPer := decimal.NewFromFloat(8)
	close := decimal.NewFromFloat(1000)
	eps := decimal.NewFromFloat(100)
	feps := decimal.NewFromFloat(125)
	bps := decimal.NewFromFloat(500)

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockValuationInteractor
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
			name: "正常系: バリュエーション指標を返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockValuationInteractor {
					m := mock_usecase.NewMockValuationInteractor(ctrl)
					m.EXPECT().GetValuation(gomock.Any(), "7203").Return(&models.Valuation{
						Symbol:      "7203",
						Close:       &close,
						PriceDate:   "2025-06-01",
						PER:         &per,
						ForwardPER:  &fwdPer,
						PBR:         &pbr,
						ROE:         &roe,
						TrailingEPS: &eps,
						ForecastEPS: &feps,
						BPS:         &bps,
						FiscalPeriod: "2025-03",
					}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/valuation?symbol=7203", nil),
			wantStatusCode: http.StatusOK,
			wantBody: &models.Valuation{
				Symbol:      "7203",
				Close:       &close,
				PriceDate:   "2025-06-01",
				PER:         &per,
				ForwardPER:  &fwdPer,
				PBR:         &pbr,
				ROE:         &roe,
				TrailingEPS: &eps,
				ForecastEPS: &feps,
				BPS:         &bps,
				FiscalPeriod: "2025-03",
			},
		},
		{
			name: "異常系: symbolが未指定",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockValuationInteractor {
					return mock_usecase.NewMockValuationInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/valuation", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは必須です\n",
		},
		{
			name: "異常系: symbolが不正文字",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockValuationInteractor {
					return mock_usecase.NewMockValuationInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("72@3")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/valuation?symbol=72@3", nil),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは英数字である必要があります\n",
		},
		{
			name: "異常系: UseCaseがエラー",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockValuationInteractor {
					m := mock_usecase.NewMockValuationInteractor(ctrl)
					m.EXPECT().GetValuation(gomock.Any(), "7203").Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("7203")
					return m
				},
			},
			req:            httptest.NewRequest(http.MethodGet, "/valuation?symbol=7203", nil),
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := NewValuationHandler(tt.fields.usecase(ctrl), tt.fields.httpServer(ctrl), zap.NewNop())
			w := httptest.NewRecorder()
			h.GetValuation(w, tt.req)

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

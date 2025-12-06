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

func TestStockPriceHandler_GetDailyPrices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// テスト用の日付
	dateStr := "2023-10-01"
	date, _ := time.Parse(util.DateLayout, dateStr)

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor
		httpServer func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBody       interface{} // 期待されるレスポンスボディ（構造体またはエラーメッセージ）
	}{
		{
			name: "正常系: 日足データを取得できる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					m := mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
					m.EXPECT().
						GetDailyStockPrices(gomock.Any(), "1234", &date, &date).
						Return([]*models.StockBrandDailyPrice{
							{
								StockBrandID: "1",
								Date:         date,
								Open:         decimal.NewFromInt(100),
								High:         decimal.NewFromInt(110),
								Low:          decimal.NewFromInt(90),
								Close:        decimal.NewFromInt(105),
								Volume:       1000,
								Adjclose:     decimal.Zero,
							},
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("1234")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=1234&from=2023-10-01&to=2023-10-01", nil),
			},
			wantStatusCode: http.StatusOK,
			wantBody: []*models.StockBrandDailyPrice{
				{
					StockBrandID: "1",
					Date:         date,
					Open:         decimal.NewFromInt(100),
					High:         decimal.NewFromInt(110),
					Low:          decimal.NewFromInt(90),
					Close:        decimal.NewFromInt(105),
					Volume:       1000,
					Adjclose:     decimal.Zero,
				},
			},
		},
		{
			name: "異常系: symbolが指定されていない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					return mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは必須です\n",
		},
		{
			name: "異常系: symbolが長すぎる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					return mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("12345678901")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=12345678901", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルが長すぎます\n",
		},
		{
			name: "異常系: symbolに不正な文字が含まれる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					return mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("1234@")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=1234@", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "シンボルは英数字である必要があります\n",
		},
		{
			name: "異常系: fromの日付フォーマットが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					return mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("1234")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=1234&from=invalid", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "fromの日付形式が不正です\n",
		},
		{
			name: "異常系: toの日付フォーマットが不正",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					return mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("1234")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(nil, errors.New("invalid date"))
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=1234&from=2023-10-01&to=invalid", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "toの日付形式が不正です\n",
		},
		{
			name: "異常系: UseCaseがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandsDailyPriceInteractor {
					m := mock_usecase.NewMockStockBrandsDailyPriceInteractor(ctrl)
					m.EXPECT().
						GetDailyStockPrices(gomock.Any(), "1234", &date, &date).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol").Return("1234")
					m.EXPECT().GetQueryParamDate(gomock.Any(), "from", util.DateLayout).Return(&date, nil)
					m.EXPECT().GetQueryParamDate(gomock.Any(), "to", util.DateLayout).Return(&date, nil)
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/daily-prices?symbol=1234&from=2023-10-01&to=2023-10-01", nil),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			u := tt.fields.usecase(ctrl)
			h := tt.fields.httpServer(ctrl)

			handler := NewStockPriceHandler(u, h, zap.NewNop())

			w := httptest.NewRecorder()
			handler.GetDailyPrices(w, tt.args.req)

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

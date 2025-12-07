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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestStockBrandHandler_GetStockBrands(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		usecase    func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor
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
		wantBody       interface{}
	}{
		{
			name: "正常系: 全件取得",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "", 0, false).
						Return(&models.PaginatedStockBrands{
							Brands: []*models.StockBrand{
								{
									ID:           "1",
									TickerSymbol: "1234",
									Name:         "テスト銘柄1",
									MarketCode:   "111",
								},
							},
							NextCursor: nil,
							Limit:      0,
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands", nil),
			},
			wantStatusCode: http.StatusOK,
			wantBody: &GetStockBrandsResponse{
				StockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1234",
						Name:         "テスト銘柄1",
						MarketCode:   "111",
					},
				},
				Pagination: nil,
			},
		},
		{
			name: "正常系: ページネーション付き取得",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					nextCursor := "5678"
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "1000", 10, false).
						Return(&models.PaginatedStockBrands{
							Brands: []*models.StockBrand{
								{
									ID:           "1",
									TickerSymbol: "1234",
									Name:         "テスト銘柄1",
									MarketCode:   "111",
								},
							},
							NextCursor: &nextCursor,
							Limit:      10,
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("1000")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("10")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?symbol_from=1000&limit=10", nil),
			},
			wantStatusCode: http.StatusOK,
			wantBody: func() *GetStockBrandsResponse {
				nextCursor := "5678"
				return &GetStockBrandsResponse{
					StockBrands: []*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
					},
					Pagination: &PaginationInfo{
						Limit:      10,
						NextCursor: &nextCursor,
					},
				}
			}(),
		},
		{
			name: "正常系: 主要市場のみフィルタリング",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "", 0, true).
						Return(&models.PaginatedStockBrands{
							Brands: []*models.StockBrand{
								{
									ID:           "1",
									TickerSymbol: "1234",
									Name:         "テスト銘柄1",
									MarketCode:   "111",
								},
							},
							NextCursor: nil,
							Limit:      0,
						}, nil)
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("true")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?only_main_markets=true", nil),
			},
			wantStatusCode: http.StatusOK,
			wantBody: &GetStockBrandsResponse{
				StockBrands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1234",
						Name:         "テスト銘柄1",
						MarketCode:   "111",
					},
				},
				Pagination: nil,
			},
		},
		{
			name: "異常系: symbol_fromが長すぎる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("12345678901")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?symbol_from=12345678901", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "symbol_fromが長すぎます\n",
		},
		{
			name: "異常系: symbol_fromに不正な文字が含まれる",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("1234@")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?symbol_from=1234@", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "symbol_fromは英数字である必要があります\n",
		},
		{
			name: "異常系: limitが数値でない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("abc")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?limit=abc", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limitは数値である必要があります\n",
		},
		{
			name: "異常系: limitが0以下",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("0")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?limit=0", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limitは正の整数である必要があります\n",
		},
		{
			name: "異常系: limitが10000を超える",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("10001")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?limit=10001", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "limitは10000以下である必要があります\n",
		},
		{
			name: "異常系: only_main_marketsがboolでない",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					return mock_usecase.NewMockStockBrandInteractor(ctrl)
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("invalid")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?only_main_markets=invalid", nil),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "only_main_marketsはtrue/falseである必要があります\n",
		},
		{
			name: "異常系: UseCaseがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "", 0, false).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands", nil),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
		{
			name: "異常系: FindAllMainMarketsがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "", 0, true).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("true")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?only_main_markets=true", nil),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
		},
		{
			name: "異常系: FindFromSymbolMainMarketsがエラーを返す",
			fields: fields{
				usecase: func(ctrl *gomock.Controller) *mock_usecase.MockStockBrandInteractor {
					m := mock_usecase.NewMockStockBrandInteractor(ctrl)
					m.EXPECT().
						GetStockBrands(gomock.Any(), "1301", 0, true).
						Return(nil, errors.New("db error"))
					return m
				},
				httpServer: func(ctrl *gomock.Controller) *mock_driver.MockHTTPServer {
					m := mock_driver.NewMockHTTPServer(ctrl)
					m.EXPECT().GetQueryParam(gomock.Any(), "symbol_from").Return("1301")
					m.EXPECT().GetQueryParam(gomock.Any(), "limit").Return("")
					m.EXPECT().GetQueryParam(gomock.Any(), "only_main_markets").Return("true")
					return m
				},
			},
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/stock-brands?symbol_from=1301&only_main_markets=true", nil),
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
			logger, _ := zap.NewDevelopment()

			handler := NewStockBrandHandler(u, h, logger)

			// ResponseRecorderを作成
			w := httptest.NewRecorder()

			// ハンドラーを実行
			handler.GetStockBrands(w, tt.args.req)

			// ステータスコードの検証
			assert.Equal(t, tt.wantStatusCode, w.Code)

			// レスポンスボディの検証
			if tt.wantStatusCode == http.StatusOK {
				var got GetStockBrandsResponse
				err := json.NewDecoder(w.Body).Decode(&got)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				assert.Equal(t, tt.wantBody, &got)
			} else {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

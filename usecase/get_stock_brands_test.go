package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestStockBrandInteractorImpl_GetStockBrands(t *testing.T) {
	type fields struct {
		stockBrandRepository func(ctrl *gomock.Controller) repositories.StockBrandRepository
	}
	type args struct {
		ctx             context.Context
		symbolFrom      string
		limit           int
		onlyMainMarkets bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.PaginatedStockBrands
		wantErr bool
	}{
		{
			name: "正常系: 全件取得",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: false,
						MarketCodes:     nil,
						SymbolFrom:      "",
						Limit:           0,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "2",
							TickerSymbol: "5678",
							Name:         "テスト銘柄2",
							MarketCode:   "112",
						},
					}, nil)
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "",
				limit:           0,
				onlyMainMarkets: false,
			},
			want: &models.PaginatedStockBrands{
				Brands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1234",
						Name:         "テスト銘柄1",
						MarketCode:   "111",
					},
					{
						ID:           "2",
						TickerSymbol: "5678",
						Name:         "テスト銘柄2",
						MarketCode:   "112",
					},
				},
				NextCursor: nil,
				Limit:      0,
			},
			wantErr: false,
		},
		{
			name: "正常系: ページネーション付き取得",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					// limit+1件取得するため11件を期待
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: false,
						MarketCodes:     nil,
						SymbolFrom:      "1000",
						Limit:           11,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "2",
							TickerSymbol: "1235",
							Name:         "テスト銘柄2",
							MarketCode:   "111",
						},
						{
							ID:           "3",
							TickerSymbol: "1236",
							Name:         "テスト銘柄3",
							MarketCode:   "111",
						},
						{
							ID:           "4",
							TickerSymbol: "1237",
							Name:         "テスト銘柄4",
							MarketCode:   "111",
						},
						{
							ID:           "5",
							TickerSymbol: "1238",
							Name:         "テスト銘柄5",
							MarketCode:   "111",
						},
						{
							ID:           "6",
							TickerSymbol: "1239",
							Name:         "テスト銘柄6",
							MarketCode:   "111",
						},
						{
							ID:           "7",
							TickerSymbol: "1240",
							Name:         "テスト銘柄7",
							MarketCode:   "111",
						},
						{
							ID:           "8",
							TickerSymbol: "1241",
							Name:         "テスト銘柄8",
							MarketCode:   "111",
						},
						{
							ID:           "9",
							TickerSymbol: "1242",
							Name:         "テスト銘柄9",
							MarketCode:   "111",
						},
						{
							ID:           "10",
							TickerSymbol: "1243",
							Name:         "テスト銘柄10",
							MarketCode:   "111",
						},
						{
							ID:           "11",
							TickerSymbol: "1244",
							Name:         "テスト銘柄11",
							MarketCode:   "111",
						},
					}, nil)
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "1000",
				limit:           10,
				onlyMainMarkets: false,
			},
			want: func() *models.PaginatedStockBrands {
				nextCursor := "1244"
				return &models.PaginatedStockBrands{
					Brands: []*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "2",
							TickerSymbol: "1235",
							Name:         "テスト銘柄2",
							MarketCode:   "111",
						},
						{
							ID:           "3",
							TickerSymbol: "1236",
							Name:         "テスト銘柄3",
							MarketCode:   "111",
						},
						{
							ID:           "4",
							TickerSymbol: "1237",
							Name:         "テスト銘柄4",
							MarketCode:   "111",
						},
						{
							ID:           "5",
							TickerSymbol: "1238",
							Name:         "テスト銘柄5",
							MarketCode:   "111",
						},
						{
							ID:           "6",
							TickerSymbol: "1239",
							Name:         "テスト銘柄6",
							MarketCode:   "111",
						},
						{
							ID:           "7",
							TickerSymbol: "1240",
							Name:         "テスト銘柄7",
							MarketCode:   "111",
						},
						{
							ID:           "8",
							TickerSymbol: "1241",
							Name:         "テスト銘柄8",
							MarketCode:   "111",
						},
						{
							ID:           "9",
							TickerSymbol: "1242",
							Name:         "テスト銘柄9",
							MarketCode:   "111",
						},
						{
							ID:           "10",
							TickerSymbol: "1243",
							Name:         "テスト銘柄10",
							MarketCode:   "111",
						},
					},
					NextCursor: &nextCursor,
					Limit:      10,
				}
			}(),
			wantErr: false,
		},
		{
			name: "正常系: 主要市場のみフィルタリング",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "",
						Limit:           0,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "3",
							TickerSymbol: "9012",
							Name:         "テスト銘柄3",
							MarketCode:   "113",
						},
					}, nil)
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "",
				limit:           0,
				onlyMainMarkets: true,
			},
			want: &models.PaginatedStockBrands{
				Brands: []*models.StockBrand{
					{
						ID:           "1",
						TickerSymbol: "1234",
						Name:         "テスト銘柄1",
						MarketCode:   "111",
					},
					{
						ID:           "3",
						TickerSymbol: "9012",
						Name:         "テスト銘柄3",
						MarketCode:   "113",
					},
				},
				NextCursor: nil,
				Limit:      0,
			},
			wantErr: false,
		},
		{
			name: "正常系: ページネーション + 主要市場フィルタリング",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					// limit+1件取得するため11件を期待
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "1000",
						Limit:           11,
					})).Return([]*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "2",
							TickerSymbol: "1235",
							Name:         "テスト銘柄2",
							MarketCode:   "112",
						},
						{
							ID:           "3",
							TickerSymbol: "1236",
							Name:         "テスト銘柄3",
							MarketCode:   "113",
						},
						{
							ID:           "4",
							TickerSymbol: "1237",
							Name:         "テスト銘柄4",
							MarketCode:   "111",
						},
						{
							ID:           "5",
							TickerSymbol: "1238",
							Name:         "テスト銘柄5",
							MarketCode:   "112",
						},
						{
							ID:           "6",
							TickerSymbol: "1239",
							Name:         "テスト銘柄6",
							MarketCode:   "113",
						},
						{
							ID:           "7",
							TickerSymbol: "1240",
							Name:         "テスト銘柄7",
							MarketCode:   "111",
						},
						{
							ID:           "8",
							TickerSymbol: "1241",
							Name:         "テスト銘柄8",
							MarketCode:   "112",
						},
						{
							ID:           "9",
							TickerSymbol: "1242",
							Name:         "テスト銘柄9",
							MarketCode:   "113",
						},
						{
							ID:           "10",
							TickerSymbol: "1243",
							Name:         "テスト銘柄10",
							MarketCode:   "111",
						},
						{
							ID:           "11",
							TickerSymbol: "1244",
							Name:         "テスト銘柄11",
							MarketCode:   "112",
						},
					}, nil)
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "1000",
				limit:           10,
				onlyMainMarkets: true,
			},
			want: func() *models.PaginatedStockBrands {
				nextCursor := "1244"
				return &models.PaginatedStockBrands{
					Brands: []*models.StockBrand{
						{
							ID:           "1",
							TickerSymbol: "1234",
							Name:         "テスト銘柄1",
							MarketCode:   "111",
						},
						{
							ID:           "2",
							TickerSymbol: "1235",
							Name:         "テスト銘柄2",
							MarketCode:   "112",
						},
						{
							ID:           "3",
							TickerSymbol: "1236",
							Name:         "テスト銘柄3",
							MarketCode:   "113",
						},
						{
							ID:           "4",
							TickerSymbol: "1237",
							Name:         "テスト銘柄4",
							MarketCode:   "111",
						},
						{
							ID:           "5",
							TickerSymbol: "1238",
							Name:         "テスト銘柄5",
							MarketCode:   "112",
						},
						{
							ID:           "6",
							TickerSymbol: "1239",
							Name:         "テスト銘柄6",
							MarketCode:   "113",
						},
						{
							ID:           "7",
							TickerSymbol: "1240",
							Name:         "テスト銘柄7",
							MarketCode:   "111",
						},
						{
							ID:           "8",
							TickerSymbol: "1241",
							Name:         "テスト銘柄8",
							MarketCode:   "112",
						},
						{
							ID:           "9",
							TickerSymbol: "1242",
							Name:         "テスト銘柄9",
							MarketCode:   "113",
						},
						{
							ID:           "10",
							TickerSymbol: "1243",
							Name:         "テスト銘柄10",
							MarketCode:   "111",
						},
					},
					NextCursor: &nextCursor,
					Limit:      10,
				}
			}(),
			wantErr: false,
		},
		{
			name: "異常系: FindWithFilterがエラーを返す（全件取得）",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: false,
						MarketCodes:     nil,
						SymbolFrom:      "",
						Limit:           0,
					})).Return(nil, errors.New("db error"))
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "",
				limit:           0,
				onlyMainMarkets: false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: FindWithFilterがエラーを返す（ページネーション）",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					// limit+1件取得するため11件を期待
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: false,
						MarketCodes:     nil,
						SymbolFrom:      "1000",
						Limit:           11,
					})).Return(nil, errors.New("db error"))
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "1000",
				limit:           10,
				onlyMainMarkets: false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: FindWithFilterがエラーを返す（主要市場）",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "",
						Limit:           0,
					})).Return(nil, errors.New("db error"))
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "",
				limit:           0,
				onlyMainMarkets: true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: FindWithFilterがエラーを返す（ページネーション+主要市場）",
			fields: fields{
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					m := mock_repositories.NewMockStockBrandRepository(ctrl)
					// limit+1件取得するため11件を期待
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(&models.StockBrandFilter{
						OnlyMainMarkets: true,
						MarketCodes:     nil,
						SymbolFrom:      "1000",
						Limit:           11,
					})).Return(nil, errors.New("db error"))
					return m
				},
			},
			args: args{
				ctx:             context.Background(),
				symbolFrom:      "1000",
				limit:           10,
				onlyMainMarkets: true,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := tt.fields.stockBrandRepository(ctrl)
			si := NewStockBrandInteractor(nil, r, nil, nil, nil, nil, nil)

			got, err := si.GetStockBrands(tt.args.ctx, tt.args.symbolFrom, tt.args.limit, tt.args.onlyMainMarkets)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockBrandInteractorImpl.GetStockBrands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

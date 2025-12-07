package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
)

func TestHighVolumeStockBrandRepositoryImpl_FindWithPagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewHighVolumeStockBrandRepositoryImpl(db)
	query := genQuery.Use(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name       string
		setup      func(t *testing.T)
		symbolFrom string
		limit      int
		wantCount  int
		wantFirst  string // First ticker symbol
		wantLast   string // Last ticker symbol
		wantErr    bool
	}{
		{
			name: "正常系: limit=0で全件取得",
			setup: func(t *testing.T) {
				// stock_brand を作成
				stockBrand1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1001",
					Name:         "Brand A",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1002",
					Name:         "Brand B",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand3 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1003",
					Name:         "Brand C",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				err := query.StockBrand.Create(stockBrand1, stockBrand2, stockBrand3)
				require.NoError(t, err)

				// high_volume_stock_brands を作成
				hvStockBrand1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand1.ID,
					TickerSymbol:  stockBrand1.TickerSymbol,
					VolumeAverage: 1000000,
					CreatedAt:     now,
				}
				hvStockBrand2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand2.ID,
					TickerSymbol:  stockBrand2.TickerSymbol,
					VolumeAverage: 2000000,
					CreatedAt:     now,
				}
				hvStockBrand3 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand3.ID,
					TickerSymbol:  stockBrand3.TickerSymbol,
					VolumeAverage: 3000000,
					CreatedAt:     now,
				}
				err = query.HighVolumeStockBrand.Create(hvStockBrand1, hvStockBrand2, hvStockBrand3)
				require.NoError(t, err)
			},
			symbolFrom: "",
			limit:      0,
			wantCount:  3,
			wantFirst:  "1001",
			wantLast:   "1003",
			wantErr:    false,
		},
		{
			name: "正常系: limitを指定して取得",
			setup: func(t *testing.T) {
				// 上記と同じセットアップ（テストごとにクリアされる）
				stockBrand1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "2001",
					Name:         "Brand D",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "2002",
					Name:         "Brand E",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand3 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "2003",
					Name:         "Brand F",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				err := query.StockBrand.Create(stockBrand1, stockBrand2, stockBrand3)
				require.NoError(t, err)

				hvStockBrand1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand1.ID,
					TickerSymbol:  stockBrand1.TickerSymbol,
					VolumeAverage: 1000000,
					CreatedAt:     now,
				}
				hvStockBrand2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand2.ID,
					TickerSymbol:  stockBrand2.TickerSymbol,
					VolumeAverage: 2000000,
					CreatedAt:     now,
				}
				hvStockBrand3 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand3.ID,
					TickerSymbol:  stockBrand3.TickerSymbol,
					VolumeAverage: 3000000,
					CreatedAt:     now,
				}
				err = query.HighVolumeStockBrand.Create(hvStockBrand1, hvStockBrand2, hvStockBrand3)
				require.NoError(t, err)
			},
			symbolFrom: "",
			limit:      2,
			wantCount:  2,
			wantFirst:  "2001",
			wantLast:   "2002",
			wantErr:    false,
		},
		{
			name: "正常系: カーソル指定で次ページ取得",
			setup: func(t *testing.T) {
				stockBrand1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "3001",
					Name:         "Brand G",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "3002",
					Name:         "Brand H",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand3 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "3003",
					Name:         "Brand I",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				err := query.StockBrand.Create(stockBrand1, stockBrand2, stockBrand3)
				require.NoError(t, err)

				hvStockBrand1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand1.ID,
					TickerSymbol:  stockBrand1.TickerSymbol,
					VolumeAverage: 1000000,
					CreatedAt:     now,
				}
				hvStockBrand2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand2.ID,
					TickerSymbol:  stockBrand2.TickerSymbol,
					VolumeAverage: 2000000,
					CreatedAt:     now,
				}
				hvStockBrand3 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand3.ID,
					TickerSymbol:  stockBrand3.TickerSymbol,
					VolumeAverage: 3000000,
					CreatedAt:     now,
				}
				err = query.HighVolumeStockBrand.Create(hvStockBrand1, hvStockBrand2, hvStockBrand3)
				require.NoError(t, err)
			},
			symbolFrom: "3001",
			limit:      2,
			wantCount:  2,
			wantFirst:  "3002",
			wantLast:   "3003",
			wantErr:    false,
		},
		{
			name:       "正常系: データが存在しない場合は空配列を返す",
			setup:      func(t *testing.T) {},
			symbolFrom: "",
			limit:      0,
			wantCount:  0,
			wantErr:    false,
		},
		{
			name: "正常系: limit=2で3件以上のデータが存在する場合、3件取得できる（次ページ判定用）",
			setup: func(t *testing.T) {
				stockBrand1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "4001",
					Name:         "Brand J",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "4002",
					Name:         "Brand K",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand3 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "4003",
					Name:         "Brand L",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand4 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "4004",
					Name:         "Brand M",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				err := query.StockBrand.Create(stockBrand1, stockBrand2, stockBrand3, stockBrand4)
				require.NoError(t, err)

				hvStockBrand1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand1.ID,
					TickerSymbol:  stockBrand1.TickerSymbol,
					VolumeAverage: 1000000,
					CreatedAt:     now,
				}
				hvStockBrand2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand2.ID,
					TickerSymbol:  stockBrand2.TickerSymbol,
					VolumeAverage: 2000000,
					CreatedAt:     now,
				}
				hvStockBrand3 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand3.ID,
					TickerSymbol:  stockBrand3.TickerSymbol,
					VolumeAverage: 3000000,
					CreatedAt:     now,
				}
				hvStockBrand4 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand4.ID,
					TickerSymbol:  stockBrand4.TickerSymbol,
					VolumeAverage: 4000000,
					CreatedAt:     now,
				}
				err = query.HighVolumeStockBrand.Create(hvStockBrand1, hvStockBrand2, hvStockBrand3, hvStockBrand4)
				require.NoError(t, err)
			},
			symbolFrom: "",
			limit:      3, // limit+1=4件取得する（実際は4件存在）
			wantCount:  3,
			wantFirst:  "4001",
			wantLast:   "4003",
			wantErr:    false,
		},
		{
			name:       "異常系: limitが負の値の場合はエラー",
			setup:      func(t *testing.T) {},
			symbolFrom: "",
			limit:      -1,
			wantCount:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テーブルをクリア
			_, err := query.HighVolumeStockBrand.WithContext(ctx).Where(query.HighVolumeStockBrand.StockBrandID.IsNotNull()).Delete()
			require.NoError(t, err)
			_, err = query.StockBrand.WithContext(ctx).Unscoped().Where(query.StockBrand.ID.IsNotNull()).Delete()
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(t)
			}

			got, err := repo.FindWithPagination(ctx, tt.symbolFrom, tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if !tt.wantErr {
				assert.Len(t, got, tt.wantCount)
				if tt.wantCount > 0 {
					assert.Equal(t, tt.wantFirst, got[0].TickerSymbol)
					assert.NotEmpty(t, got[0].CompanyName) // JOINで銘柄名が取得できていることを確認
					assert.Equal(t, tt.wantLast, got[len(got)-1].TickerSymbol)
				}
			}
		})
	}
}

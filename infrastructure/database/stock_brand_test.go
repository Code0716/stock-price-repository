package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
)

func TestStockBrandRepositoryImpl_UpsertStockBrands(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second) // MySQLのdatetime精度に合わせる

	tests := []struct {
		name    string
		input   []*models.StockBrand
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "新規作成_正常系",
			input: []*models.StockBrand{
				{
					ID:               "uuid-1",
					TickerSymbol:     "1001",
					Name:             "Test Brand 1",
					MarketCode:       "111",
					MarketName:       "Prime",
					Sector33Code:     "0050",
					Sector33CodeName: "Fishery",
					Sector17Code:     "1",
					Sector17CodeName: "FOODS",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
			wantErr: false,
			check: func(t *testing.T) {
				brands, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, brands, 1)
				assert.Equal(t, "Test Brand 1", brands[0].Name)
			},
		},
		{
			name: "更新_正常系",
			input: []*models.StockBrand{
				{
					ID:               "uuid-1",
					TickerSymbol:     "1001",
					Name:             "Test Brand 1 Updated", // 名前を変更
					MarketCode:       "111",
					MarketName:       "Prime",
					Sector33Code:     "0050",
					Sector33CodeName: "Fishery",
					Sector17Code:     "1",
					Sector17CodeName: "FOODS",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
			wantErr: false,
			check: func(t *testing.T) {
				brands, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, brands, 1)
				assert.Equal(t, "Test Brand 1 Updated", brands[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpsertStockBrands(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

func TestStockBrandRepositoryImpl_FindAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()

	// 初期データ投入
	initialBrands := []*models.StockBrand{
		{
			ID:           "uuid-1",
			TickerSymbol: "1001",
			Name:         "Brand 1",
			MarketCode:   "111",
			MarketName:   "Prime",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           "uuid-2",
			TickerSymbol: "1002",
			Name:         "Brand 2",
			MarketCode:   "112",
			MarketName:   "Standard",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
	err := repo.UpsertStockBrands(ctx, initialBrands)
	require.NoError(t, err)

	tests := []struct {
		name    string
		wantLen int
		wantErr bool
	}{
		{
			name:    "全件取得_正常系",
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindAll(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

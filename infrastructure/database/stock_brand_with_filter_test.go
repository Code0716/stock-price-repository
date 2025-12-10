package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
)

func TestStockBrandRepositoryImpl_FindWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// テスト用データ投入
	initialBrands := []*models.StockBrand{
		{
			ID:               "uuid-1",
			TickerSymbol:     "1001",
			Name:             "Brand Prime 1",
			MarketCode:       "111", // Prime
			MarketName:       "Prime",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery",
			Sector17Code:     "1",
			Sector17CodeName: "FOODS",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "uuid-2",
			TickerSymbol:     "1002",
			Name:             "Brand Standard 1",
			MarketCode:       "112", // Standard
			MarketName:       "Standard",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery",
			Sector17Code:     "1",
			Sector17CodeName: "FOODS",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "uuid-3",
			TickerSymbol:     "1003",
			Name:             "Brand Growth 1",
			MarketCode:       "113", // Growth
			MarketName:       "Growth",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery",
			Sector17Code:     "1",
			Sector17CodeName: "FOODS",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "uuid-4",
			TickerSymbol:     "1004",
			Name:             "Brand Other 1",
			MarketCode:       "999", // その他の市場
			MarketName:       "Other",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery",
			Sector17Code:     "1",
			Sector17CodeName: "FOODS",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "uuid-5",
			TickerSymbol:     "1005",
			Name:             "Brand Prime 2",
			MarketCode:       "111", // Prime
			MarketName:       "Prime",
			Sector33Code:     "0050",
			Sector33CodeName: "Fishery",
			Sector17Code:     "1",
			Sector17CodeName: "FOODS",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	err := repo.UpsertStockBrands(ctx, initialBrands)
	require.NoError(t, err)

	tests := []struct {
		name       string
		filter     *models.StockBrandFilter
		wantLen    int
		wantErr    bool
		checkFirst func(t *testing.T, brands []*models.StockBrand)
	}{
		{
			name:    "フィルタなし_全件取得",
			filter:  models.NewStockBrandFilter(),
			wantLen: 5,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1001", brands[0].TickerSymbol)
			},
		},
		{
			name:    "主要市場のみ取得",
			filter:  models.NewStockBrandFilter().WithOnlyMainMarkets(),
			wantLen: 4,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1001", brands[0].TickerSymbol)
				// 全てのブランドが主要市場であることを確認
				for _, b := range brands {
					assert.Contains(t, []string{"111", "112", "113"}, b.MarketCode)
				}
			},
		},
		{
			name:    "特定市場コード指定_Prime",
			filter:  models.NewStockBrandFilter().WithMarketCodes("111"),
			wantLen: 2,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "111", brands[0].MarketCode)
				assert.Equal(t, "111", brands[1].MarketCode)
			},
		},
		{
			name:    "特定市場コード指定_StandardとGrowth",
			filter:  models.NewStockBrandFilter().WithMarketCodes("112", "113"),
			wantLen: 2,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				// 全てのブランドがStandardまたはGrowthであることを確認
				for _, b := range brands {
					assert.Contains(t, []string{"112", "113"}, b.MarketCode)
				}
			},
		},
		{
			name:    "ページネーション_symbolFrom指定",
			filter:  models.NewStockBrandFilter().WithPagination("1002", 0),
			wantLen: 3,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1003", brands[0].TickerSymbol)
			},
		},
		{
			name:    "ページネーション_limit指定",
			filter:  models.NewStockBrandFilter().WithPagination("", 2),
			wantLen: 2,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1001", brands[0].TickerSymbol)
				assert.Equal(t, "1002", brands[1].TickerSymbol)
			},
		},
		{
			name:    "ページネーション_symbolFromとlimit両方指定",
			filter:  models.NewStockBrandFilter().WithPagination("1001", 2),
			wantLen: 2,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1002", brands[0].TickerSymbol)
				assert.Equal(t, "1003", brands[1].TickerSymbol)
			},
		},
		{
			name: "主要市場とページネーション組み合わせ",
			filter: models.NewStockBrandFilter().
				WithOnlyMainMarkets().
				WithPagination("1002", 2),
			wantLen: 2,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				assert.Equal(t, "1003", brands[0].TickerSymbol)
				assert.Equal(t, "1005", brands[1].TickerSymbol)
				// 全てのブランドが主要市場であることを確認
				for _, b := range brands {
					assert.Contains(t, []string{"111", "112", "113"}, b.MarketCode)
				}
			},
		},
		{
			name: "主要市場フラグとMarketCodes両方指定_主要市場フラグが優先される",
			filter: models.NewStockBrandFilter().
				WithOnlyMainMarkets().
				WithMarketCodes("999"), // この指定は無視される
			wantLen: 4,
			wantErr: false,
			checkFirst: func(t *testing.T, brands []*models.StockBrand) {
				// 全てのブランドが主要市場であることを確認
				for _, b := range brands {
					assert.Contains(t, []string{"111", "112", "113"}, b.MarketCode)
				}
			},
		},
		{
			name:    "該当データなし_symbolFromが最大値より大きい",
			filter:  models.NewStockBrandFilter().WithPagination("9999", 0),
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindWithFilter(ctx, tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				if tt.checkFirst != nil && len(got) > 0 {
					tt.checkFirst(t, got)
				}
			}
		})
	}
}

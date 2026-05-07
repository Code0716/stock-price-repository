package database

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Code0716/stock-price-repository/models"
)

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_FindWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(24 * time.Hour)

	stockBrand := &models.StockBrand{
		ID:           "brand-find-1",
		TickerSymbol: "2001",
		Name:         "Find Test Brand",
		MarketCode:   "111",
		MarketName:   "Prime",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err := stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{stockBrand})
	require.NoError(t, err)

	err = db.Exec(
		"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"find-uuid-1", "brand-find-1", "2001", 1000.0, "Buy", "analyze_stock_brand_price_by_sector: 25日", now.Format("2006-01-02"),
	).Error
	require.NoError(t, err)

	t.Run("daily_price なし: current_price が trade_price にフォールバックし差額が 0", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "2001",
		})
		require.NoError(t, err)
		require.Len(t, results, 1)

		assert.Equal(t, decimal.NewFromFloat(1000.0), results[0].TradePrice)
		assert.Equal(t, decimal.NewFromFloat(1000.0), results[0].CurrentPrice)
		assert.Equal(t, decimal.NewFromFloat(0), results[0].PriceDifference)
	})

	t.Run("daily_price あり: close_price が current_price になり差額が反映される", func(t *testing.T) {
		dailyPriceID := "daily-find-1"
		err = db.Exec(
			`INSERT INTO stock_brands_daily_price (id, stock_brand_id, ticker_symbol, date, open_price, close_price, high_price, low_price, adj_close_price, volume) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			dailyPriceID, "brand-find-1", "2001", now.Format("2006-01-02"), 1100.0, 1200.0, 1250.0, 1050.0, 1200.0, 10000,
		).Error
		require.NoError(t, err)
		defer db.Exec("DELETE FROM stock_brands_daily_price WHERE id = ?", dailyPriceID)

		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "2001",
		})
		require.NoError(t, err)
		require.Len(t, results, 1)

		assert.Equal(t, decimal.NewFromFloat(1000.0), results[0].TradePrice)
		assert.Equal(t, decimal.NewFromFloat(1200.0), results[0].CurrentPrice)
		assert.Equal(t, decimal.NewFromFloat(200.0), results[0].PriceDifference)
	})
}

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_DeleteByStockBrandIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(24 * time.Hour)

	stockBrands := []*models.StockBrand{
		{
			ID:           "brand-del-1",
			TickerSymbol: "3001",
			Name:         "Delete Test Brand 1",
			MarketCode:   "111",
			MarketName:   "Prime",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "brand-del-2",
			TickerSymbol: "3002",
			Name:         "Delete Test Brand 2",
			MarketCode:   "111",
			MarketName:   "Prime",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err := stockBrandRepo.UpsertStockBrands(ctx, stockBrands)
	require.NoError(t, err)

	err = db.Exec(
		"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"del-uuid-1", "brand-del-1", "3001", 1000.0, "Buy", "analyze_stock_brand_price_by_sector: 25日", now.Format("2006-01-02"),
	).Error
	require.NoError(t, err)

	err = db.Exec(
		"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"del-uuid-2", "brand-del-2", "3002", 2000.0, "Buy", "analyze_stock_brand_price_by_sector: 25日", now.Format("2006-01-02"),
	).Error
	require.NoError(t, err)

	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{
			name:    "削除_正常系",
			ids:     []string{"brand-del-1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.DeleteByStockBrandIDs(ctx, tt.ids)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

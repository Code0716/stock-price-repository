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

		assert.True(t, results[0].TradePrice.Equal(decimal.NewFromFloat(1000.0)))
		assert.True(t, results[0].CurrentPrice.Equal(decimal.NewFromFloat(1000.0)))
		assert.True(t, results[0].PriceDifference.IsZero())
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

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_FindWithFilter_Pagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	base := time.Now().Truncate(24 * time.Hour)

	brand := &models.StockBrand{
		ID:           "brand-page-1",
		TickerSymbol: "9001",
		Name:         "Page Test Brand",
		MarketCode:   "111",
		MarketName:   "Prime",
		CreatedAt:    base,
		UpdatedAt:    base,
	}
	require.NoError(t, stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{brand}))

	// 5件挿入: created_at を1日ずつずらす
	for i := 0; i < 5; i++ {
		d := base.AddDate(0, 0, -i)
		err := db.Exec(
			"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"page-uuid-"+string(rune('1'+i)), "brand-page-1", "9001", float64(1000+i*100), "Buy", "analyze_stock_brand_price_by_sector: 25日", d.Format("2006-01-02"),
		).Error
		require.NoError(t, err)
	}

	t.Run("page=1 limit=2 で最新2件取得", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "9001",
			Page:         1,
			Limit:        2,
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("page=2 limit=2 で次の2件取得", func(t *testing.T) {
		p1, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "9001",
			Page:         1,
			Limit:        2,
		})
		require.NoError(t, err)

		p2, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "9001",
			Page:         2,
			Limit:        2,
		})
		require.NoError(t, err)
		assert.Len(t, p2, 2)

		// 重複なし
		p1IDs := map[string]bool{p1[0].ID: true, p1[1].ID: true}
		for _, h := range p2 {
			assert.False(t, p1IDs[h.ID], "page1とpage2のIDが重複している: %s", h.ID)
		}
	})

	t.Run("created_at ASC ソート", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "9001",
			SortBy:       models.AnalyzeStockBrandPriceHistorySortByCreatedAt,
			Order:        models.AnalyzeStockBrandPriceHistoryOrderAsc,
			Page:         1,
			Limit:        5,
		})
		require.NoError(t, err)
		require.Len(t, results, 5)
		for i := 1; i < len(results); i++ {
			assert.True(t, !results[i].CreatedAt.Before(results[i-1].CreatedAt), "ASCソートが崩れている")
		}
	})

	t.Run("created_at DESC ソート (デフォルト)", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "9001",
			SortBy:       models.AnalyzeStockBrandPriceHistorySortByCreatedAt,
			Order:        models.AnalyzeStockBrandPriceHistoryOrderDesc,
			Page:         1,
			Limit:        5,
		})
		require.NoError(t, err)
		require.Len(t, results, 5)
		for i := 1; i < len(results); i++ {
			assert.True(t, !results[i].CreatedAt.After(results[i-1].CreatedAt), "DESCソートが崩れている")
		}
	})
}

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_CountWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(24 * time.Hour)

	brands := []*models.StockBrand{
		{ID: "brand-cnt-1", TickerSymbol: "8001", Name: "Count Brand 1", MarketCode: "111", MarketName: "Prime", CreatedAt: now, UpdatedAt: now},
		{ID: "brand-cnt-2", TickerSymbol: "8002", Name: "Count Brand 2", MarketCode: "111", MarketName: "Prime", CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, stockBrandRepo.UpsertStockBrands(ctx, brands))

	inserts := []struct {
		id     string
		brand  string
		ticker string
		action string
		method string
	}{
		{"cnt-1", "brand-cnt-1", "8001", "Buy", "analyze_stock_brand_price_by_sector: 25日"},
		{"cnt-2", "brand-cnt-1", "8001", "Sell", "analyze_stock_brand_price_by_sector: 75日"},
		{"cnt-3", "brand-cnt-2", "8002", "Buy", "analyze_stock_brand_price_by_sector: 25日"},
	}
	for _, ins := range inserts {
		require.NoError(t, db.Exec(
			"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			ins.id, ins.brand, ins.ticker, 1000.0, ins.action, ins.method, now.Format("2006-01-02"),
		).Error)
	}

	tests := []struct {
		name   string
		filter *models.AnalyzeStockBrandPriceHistoryFilter
		want   int64
	}{
		{
			name:   "フィルタなし: 全件",
			filter: &models.AnalyzeStockBrandPriceHistoryFilter{},
			want:   3,
		},
		{
			name:   "ticker=8001 でフィルタ",
			filter: &models.AnalyzeStockBrandPriceHistoryFilter{TickerSymbol: "8001"},
			want:   2,
		},
		{
			name:   "action=Buy でフィルタ",
			filter: &models.AnalyzeStockBrandPriceHistoryFilter{Action: "Buy"},
			want:   2,
		},
		{
			name:   "ticker=8001 + action=Sell",
			filter: &models.AnalyzeStockBrandPriceHistoryFilter{TickerSymbol: "8001", Action: "Sell"},
			want:   1,
		},
		{
			name:   "存在しないticker",
			filter: &models.AnalyzeStockBrandPriceHistoryFilter{TickerSymbol: "9999"},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := repo.CountWithFilter(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, count)
		})
	}
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

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_FindWithFilter_SortByProfit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(24 * time.Hour)

	brand := &models.StockBrand{
		ID:           "brand-profit-1",
		TickerSymbol: "7001",
		Name:         "Profit Brand",
		MarketCode:   "111",
		MarketName:   "Prime",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{brand}))

	// 3件: trade_price が異なる（daily_price なし → current_price = trade_price → profit = 0）
	// profit を確認するには daily_price が必要だが、sort自体は実行できることを確認する
	for i, price := range []float64{1000.0, 2000.0, 500.0} {
		require.NoError(t, db.Exec(
			"INSERT INTO analyze_stock_brand_price_history (id, stock_brand_id, ticker_symbol, trade_price, action, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"profit-uuid-"+string(rune('1'+i)), "brand-profit-1", "7001", price, "Buy", "find_macd_bullish_stock_v1", now.AddDate(0, 0, -i).Format("2006-01-02"),
		).Error)
	}

	t.Run("sort_by=profit DESC でエラーなく取得できる", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "7001",
			SortBy:       models.AnalyzeStockBrandPriceHistorySortByProfit,
			Order:        models.AnalyzeStockBrandPriceHistoryOrderDesc,
			Page:         1,
			Limit:        10,
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("sort_by=profit_rate ASC でエラーなく取得できる", func(t *testing.T) {
		results, err := repo.FindWithFilter(ctx, &models.AnalyzeStockBrandPriceHistoryFilter{
			TickerSymbol: "7001",
			SortBy:       models.AnalyzeStockBrandPriceHistorySortByProfitRate,
			Order:        models.AnalyzeStockBrandPriceHistoryOrderAsc,
			Page:         1,
			Limit:        10,
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}

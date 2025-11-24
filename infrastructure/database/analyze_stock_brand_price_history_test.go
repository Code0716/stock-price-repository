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

func stringPtr(s string) *string {
	return &s
}

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_CreateOrUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 親となるStockBrandを作成
	stockBrand := &models.StockBrand{
		ID:           "brand-1",
		TickerSymbol: "1001",
		Name:         "Test Brand 1",
		MarketCode:   "111",
		MarketName:   "Prime",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err := stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{stockBrand})
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   []*models.AnalyzeStockBrandPriceHistory
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "新規作成_正常系",
			input: []*models.AnalyzeStockBrandPriceHistory{
				{
					ID:           "uuid-1",
					StockBrandID: "brand-1",
					TickerSymbol: "1001",
					TradePrice:   decimal.NewFromFloat(1000),
					CurrentPrice: decimal.NewFromFloat(1000),
					Action:       "buy",
					Method:       "method1",
					Memo:         stringPtr("memo1"),
					CreatedAt:    now,
				},
			},
			wantErr: false,
			check: func(_ *testing.T) {
				// 確認用のFindメソッドがないため、直接DBを確認するか、エラーがないことで判断
				// ここではエラーがないことと、DeleteByStockBrandIDsで削除できることで間接的に確認
			},
		},
		{
			name: "更新_正常系",
			input: []*models.AnalyzeStockBrandPriceHistory{
				{
					ID:           "uuid-1",
					StockBrandID: "brand-1",
					TickerSymbol: "1001",
					TradePrice:   decimal.NewFromFloat(1000),
					CurrentPrice: decimal.NewFromFloat(1200), // 価格更新
					Action:       "buy",
					Method:       "method1",
					Memo:         stringPtr("memo1"),
					CreatedAt:    now,
				},
			},
			wantErr: false,
			check: func(_ *testing.T) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrUpdate(ctx, tt.input)
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

func TestAnalyzeStockBrandPriceHistoryRepositoryImpl_DeleteByStockBrandIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 親となるStockBrandを作成
	stockBrands := []*models.StockBrand{
		{
			ID:           "brand-1",
			TickerSymbol: "1001",
			Name:         "Test Brand 1",
			MarketCode:   "111",
			MarketName:   "Prime",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "brand-2",
			TickerSymbol: "1002",
			Name:         "Test Brand 2",
			MarketCode:   "111",
			MarketName:   "Prime",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err := stockBrandRepo.UpsertStockBrands(ctx, stockBrands)
	require.NoError(t, err)

	// 初期データ投入
	initialData := []*models.AnalyzeStockBrandPriceHistory{
		{
			ID:           "uuid-1",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			TradePrice:   decimal.NewFromFloat(1000),
			CurrentPrice: decimal.NewFromFloat(1000),
			CreatedAt:    now,
		},
		{
			ID:           "uuid-2",
			StockBrandID: "brand-2",
			TickerSymbol: "1002",
			TradePrice:   decimal.NewFromFloat(2000),
			CurrentPrice: decimal.NewFromFloat(2000),
			CreatedAt:    now,
		},
	}
	err = repo.CreateOrUpdate(ctx, initialData)
	require.NoError(t, err)

	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{
			name:    "削除_正常系",
			ids:     []string{"brand-1"},
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

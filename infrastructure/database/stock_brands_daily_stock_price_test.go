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

func TestStockBrandsDailyPriceRepositoryImpl_CreateStockBrandDailyPrice(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceRepositoryImpl(db)
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
		input   []*models.StockBrandDailyPrice
		wantErr bool
	}{
		{
			name: "新規作成_正常系",
			input: []*models.StockBrandDailyPrice{
				{
					ID:           "uuid-1",
					StockBrandID: "brand-1",
					TickerSymbol: "1001",
					Date:         now,
					Open:         decimal.NewFromFloat(1000),
					Close:        decimal.NewFromFloat(1000),
					High:         decimal.NewFromFloat(1000),
					Low:          decimal.NewFromFloat(1000),
					Adjclose:     decimal.NewFromFloat(1000),
					Volume:       1000,
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateStockBrandDailyPrice(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStockBrandsDailyPriceRepositoryImpl_GetLatestPriceBySymbol(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

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

	// 初期データ投入
	initialData := []*models.StockBrandDailyPrice{
		{
			ID:           "uuid-1",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			Date:         yesterday,
			Open:         decimal.NewFromFloat(1000),
			Close:        decimal.NewFromFloat(1000),
			High:         decimal.NewFromFloat(1000),
			Low:          decimal.NewFromFloat(1000),
			Adjclose:     decimal.NewFromFloat(1000),
			Volume:       1000,
			CreatedAt:    yesterday,
			UpdatedAt:    yesterday,
		},
		{
			ID:           "uuid-2",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			Date:         today, // 最新
			Open:         decimal.NewFromFloat(1100),
			Close:        decimal.NewFromFloat(1100),
			High:         decimal.NewFromFloat(1100),
			Low:          decimal.NewFromFloat(1100),
			Adjclose:     decimal.NewFromFloat(1100),
			Volume:       1100,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err = repo.CreateStockBrandDailyPrice(ctx, initialData)
	require.NoError(t, err)

	tests := []struct {
		name    string
		symbol  string
		want    *models.StockBrandDailyPrice
		wantErr bool
	}{
		{
			name:   "最新価格取得_正常系",
			symbol: "1001",
			want: &models.StockBrandDailyPrice{
				TickerSymbol: "1001",
				Date:         today,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetLatestPriceBySymbol(ctx, tt.symbol)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.TickerSymbol, got.TickerSymbol)
				// assert.True(t, tt.want.Date.Equal(got.Date) || tt.want.Date.Sub(got.Date) < time.Second)
				assert.WithinDuration(t, tt.want.Date, got.Date, time.Second)
			}
		})
	}
}

func TestStockBrandsDailyPriceRepositoryImpl_ListDailyPricesBySymbol(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

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

	// 初期データ投入
	initialData := []*models.StockBrandDailyPrice{
		{
			ID:           "uuid-1",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			Date:         yesterday,
			Open:         decimal.NewFromFloat(1000),
			Close:        decimal.NewFromFloat(1000),
			High:         decimal.NewFromFloat(1000),
			Low:          decimal.NewFromFloat(1000),
			Adjclose:     decimal.NewFromFloat(1000),
			Volume:       1000,
			CreatedAt:    yesterday,
			UpdatedAt:    yesterday,
		},
		{
			ID:           "uuid-2",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			Date:         today,
			Open:         decimal.NewFromFloat(1100),
			Close:        decimal.NewFromFloat(1100),
			High:         decimal.NewFromFloat(1100),
			Low:          decimal.NewFromFloat(1100),
			Adjclose:     decimal.NewFromFloat(1100),
			Volume:       1100,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err = repo.CreateStockBrandDailyPrice(ctx, initialData)
	require.NoError(t, err)

	tests := []struct {
		name    string
		filter  models.ListDailyPricesBySymbolFilter
		wantLen int
		wantErr bool
	}{
		{
			name: "全件取得_正常系",
			filter: models.ListDailyPricesBySymbolFilter{
				TickerSymbol: "1001",
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "日付指定_正常系",
			filter: models.ListDailyPricesBySymbolFilter{
				TickerSymbol: "1001",
				DateFrom:     &today,
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ListDailyPricesBySymbol(ctx, tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

func TestStockBrandsDailyPriceRepositoryImpl_DeleteByIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceRepositoryImpl(db)
	stockBrandRepo := NewStockBrandRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 親レコード(StockBrand)を作成
	stockBrand := &models.StockBrand{
		ID:           "brand-1",
		TickerSymbol: "1001",
		Name:         "Test Brand",
	}
	err := stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{stockBrand})
	require.NoError(t, err)

	// 初期データ投入
	initialData := []*models.StockBrandDailyPrice{
		{
			ID:           "uuid-1",
			StockBrandID: "brand-1",
			TickerSymbol: "1001",
			Date:         now,
			Open:         decimal.NewFromFloat(1000),
			Close:        decimal.NewFromFloat(1000),
			High:         decimal.NewFromFloat(1000),
			Low:          decimal.NewFromFloat(1000),
			Adjclose:     decimal.NewFromFloat(1000),
			Volume:       1000,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err = repo.CreateStockBrandDailyPrice(ctx, initialData)
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
			err := repo.DeleteByIDs(ctx, tt.ids)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

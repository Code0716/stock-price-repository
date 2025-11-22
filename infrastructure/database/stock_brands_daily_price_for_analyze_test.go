package database

import (
	"context"
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStockBrandsDailyPriceForAnalyzeRepositoryImpl_CreateStockBrandDailyPriceForAnalyze(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name    string
		input   []*models.StockBrandDailyPriceForAnalyze
		wantErr bool
	}{
		{
			name: "新規作成_正常系",
			input: []*models.StockBrandDailyPriceForAnalyze{
				{
					ID:           "uuid-1",
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
			err := repo.CreateStockBrandDailyPriceForAnalyze(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStockBrandsDailyPriceForAnalyzeRepositoryImpl_ListLatestPriceBySymbols(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	yesterday := now.AddDate(0, 0, -1)

	// 初期データ投入
	initialData := []*models.StockBrandDailyPriceForAnalyze{
		{
			ID:           "uuid-1",
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
			TickerSymbol: "1001",
			Date:         now, // 最新
			Open:         decimal.NewFromFloat(1100),
			Close:        decimal.NewFromFloat(1100),
			High:         decimal.NewFromFloat(1100),
			Low:          decimal.NewFromFloat(1100),
			Adjclose:     decimal.NewFromFloat(1100),
			Volume:       1100,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "uuid-3",
			TickerSymbol: "1002",
			Date:         now,
			Open:         decimal.NewFromFloat(2000),
			Close:        decimal.NewFromFloat(2000),
			High:         decimal.NewFromFloat(2000),
			Low:          decimal.NewFromFloat(2000),
			Adjclose:     decimal.NewFromFloat(2000),
			Volume:       2000,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	err := repo.CreateStockBrandDailyPriceForAnalyze(ctx, initialData)
	require.NoError(t, err)

	tests := []struct {
		name    string
		symbols []*string
		wantLen int
		wantErr bool
	}{
		{
			name:    "最新価格取得_正常系",
			symbols: []*string{util.ToPtrGenerics("1001"), util.ToPtrGenerics("1002")},
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ListLatestPriceBySymbols(ctx, tt.symbols)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				// 1001の最新価格が取得できているか確認
				for _, p := range got {
					if p.TickerSymbol == "1001" {
						assert.True(t, p.Date.Equal(now) || p.Date.Sub(now) < time.Second)
					}
				}
			}
		})
	}
}

func TestStockBrandsDailyPriceForAnalyzeRepositoryImpl_DeleteBySymbols(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 初期データ投入
	initialData := []*models.StockBrandDailyPriceForAnalyze{
		{
			ID:           "uuid-1",
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
	err := repo.CreateStockBrandDailyPriceForAnalyze(ctx, initialData)
	require.NoError(t, err)

	tests := []struct {
		name    string
		symbols []string
		wantErr bool
	}{
		{
			name:    "削除_正常系",
			symbols: []string{"1001"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.DeleteBySymbols(ctx, tt.symbols)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

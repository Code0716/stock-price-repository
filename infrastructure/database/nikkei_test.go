package database

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func TestNikkeiRepositoryImpl_CreateNikkeiStockAverageDailyPrices(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewNikkeiRepositoryImpl(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name    string
		input   models.IndexStockAverageDailyPrices
		wantErr bool
	}{
		{
			name: "新規作成_正常系",
			input: models.IndexStockAverageDailyPrices{
				{
					Date:      now,
					Open:      decimal.NewFromFloat(28000.0),
					Close:     decimal.NewFromFloat(28100.0),
					High:      decimal.NewFromFloat(28200.0),
					Low:       decimal.NewFromFloat(27900.0),
					Adjclose:  decimal.NewFromFloat(28100.0),
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			wantErr: false,
		},
		{
			name: "更新_正常系",
			input: models.IndexStockAverageDailyPrices{
				{
					Date:      now,
					Open:      decimal.NewFromFloat(28000.0),
					Close:     decimal.NewFromFloat(28500.0), // 更新
					High:      decimal.NewFromFloat(28600.0),
					Low:       decimal.NewFromFloat(27900.0),
					Adjclose:  decimal.NewFromFloat(28500.0),
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateNikkeiStockAverageDailyPrices(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

package database

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func TestDjiRepositoryImpl_CreateDjiStockAverageDailyPrices(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewDjiRepositoryImpl(db)
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
					Open:      decimal.NewFromFloat(30000.0),
					Close:     decimal.NewFromFloat(30100.0),
					High:      decimal.NewFromFloat(30200.0),
					Low:       decimal.NewFromFloat(29900.0),
					Adjclose:  decimal.NewFromFloat(30100.0),
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
					Open:      decimal.NewFromFloat(30000.0),
					Close:     decimal.NewFromFloat(30500.0), // 更新
					High:      decimal.NewFromFloat(30600.0),
					Low:       decimal.NewFromFloat(29900.0),
					Adjclose:  decimal.NewFromFloat(30500.0),
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateDjiStockAverageDailyPrices(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

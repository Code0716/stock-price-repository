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

func TestAppliedStockConsolidationsHistoryRepositoryImpl_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAppliedStockConsolidationsHistoryRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		input   *models.AppliedStockConsolidationHistory
		wantErr bool
	}{
		{
			name: "正常作成",
			input: &models.AppliedStockConsolidationHistory{
				Symbol:            "1234",
				ConsolidationDate: today,
				Ratio:             decimal.NewFromFloat(5.0),
				AppliedAt:         today,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Existsで登録を相互検証
				exists, err := repo.Exists(ctx, tt.input.Symbol, tt.input.ConsolidationDate)
				require.NoError(t, err)
				assert.True(t, exists, "Createしたデータが存在すること")
			}
		})
	}
}

func TestAppliedStockConsolidationsHistoryRepositoryImpl_Exists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAppliedStockConsolidationsHistoryRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// 事前データ投入
	initialData := &models.AppliedStockConsolidationHistory{
		Symbol:            "5678",
		ConsolidationDate: today,
		Ratio:             decimal.NewFromFloat(2.5),
		AppliedAt:         today,
	}
	err := repo.Create(ctx, initialData)
	require.NoError(t, err)

	type args struct {
		symbol            string
		consolidationDate time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "データが存在する場合_Trueを返す",
			args: args{
				symbol:            "5678",
				consolidationDate: today,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "シンボルが異なる場合_Falseを返す",
			args: args{
				symbol:            "9999",
				consolidationDate: today,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "日付が異なる場合_Falseを返す",
			args: args{
				symbol:            "5678",
				consolidationDate: today.Add(24 * time.Hour),
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Exists(ctx, tt.args.symbol, tt.args.consolidationDate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

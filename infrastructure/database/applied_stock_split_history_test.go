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

func TestAppliedStockSplitsHistoryRepositoryImpl_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAppliedStockSplitsHistoryRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		input   *models.AppliedStockSplitHistory
		wantErr bool
	}{
		{
			name: "正常作成",
			input: &models.AppliedStockSplitHistory{
				Symbol:    "1234",
				SplitDate: today,
				Ratio:     decimal.NewFromFloat(2.0),
				AppliedAt: today,
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

				// 検証: Existsメソッドを使って登録を確認、または直接DBクエリで確認したいところだが、
				// ここでは単体テストなのでExistsメソッドのテストは別に行い、ここではCreateのエラー有無と
				// 相互検証的にExistsがTrueになることを確認する。
				exists, err := repo.Exists(ctx, tt.input.Symbol, tt.input.SplitDate)
				require.NoError(t, err)
				assert.True(t, exists, "Createしたデータが存在すること")
			}
		})
	}
}

func TestAppliedStockSplitsHistoryRepositoryImpl_Exists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAppliedStockSplitsHistoryRepositoryImpl(db)
	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// 事前データ投入
	initialData := &models.AppliedStockSplitHistory{
		Symbol:    "5678",
		SplitDate: today,
		Ratio:     decimal.NewFromFloat(1.5),
		AppliedAt: today,
	}
	err := repo.Create(ctx, initialData)
	require.NoError(t, err)

	type args struct {
		symbol    string
		splitDate time.Time
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
				symbol:    "5678",
				splitDate: today,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "シンボルが異なる場合_Falseを返す",
			args: args{
				symbol:    "9999",
				splitDate: today,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "日付が異なる場合_Falseを返す",
			args: args{
				symbol:    "5678",
				splitDate: today.Add(24 * time.Hour),
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Exists(ctx, tt.args.symbol, tt.args.splitDate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

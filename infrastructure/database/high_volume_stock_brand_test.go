package database

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
)

func TestHighVolumeStockBrandRepositoryImpl_FindAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewHighVolumeStockBrandRepositoryImpl(db)
	query := genQuery.Use(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name    string
		setup   func(t *testing.T)
		want    []*models.HighVolumeStockBrand
		wantErr bool
	}{
		{
			name: "正常系: 全件取得成功",
			setup: func(t *testing.T) {
				// stock_brand を作成
				stockBrand1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1001",
					Name:         "Test Brand 1",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				stockBrand2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1002",
					Name:         "Test Brand 2",
					MarketCode:   "111",
					MarketName:   "Prime",
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				err := query.StockBrand.Create(stockBrand1, stockBrand2)
				require.NoError(t, err)

				// high_volume_stock_brands を作成
				hvStockBrand1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand1.ID,
					TickerSymbol:  stockBrand1.TickerSymbol,
					VolumeAverage: 1000000,
					CreatedAt:     now,
				}
				hvStockBrand2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  stockBrand2.ID,
					TickerSymbol:  stockBrand2.TickerSymbol,
					VolumeAverage: 2000000,
					CreatedAt:     now,
				}
				err = query.HighVolumeStockBrand.Create(hvStockBrand1, hvStockBrand2)
				require.NoError(t, err)
			},
			want: []*models.HighVolumeStockBrand{
				// 結果はテスト内で検証
			},
			wantErr: false,
		},
		{
			name:    "正常系: データが存在しない場合は空配列を返す",
			setup:   func(t *testing.T) {},
			want:    []*models.HighVolumeStockBrand{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テーブルをクリア
			_, err := query.HighVolumeStockBrand.WithContext(ctx).Where(query.HighVolumeStockBrand.StockBrandID.IsNotNull()).Delete()
			require.NoError(t, err)
			_, err = query.StockBrand.WithContext(ctx).Unscoped().Where(query.StockBrand.ID.IsNotNull()).Delete()
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(t)
			}

			got, err := repo.FindAll(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if !tt.wantErr && tt.name == "正常系: データが存在しない場合は空配列を返す" {
				assert.Len(t, got, 0)
			} else if !tt.wantErr && tt.name == "正常系: 全件取得成功" {
				assert.Len(t, got, 2)
				// TickerSymbolでソート（データベースの返却順序は保証されないため）
				if got[0].TickerSymbol == "1002" {
					got[0], got[1] = got[1], got[0]
				}
				assert.Equal(t, "1001", got[0].TickerSymbol)
				assert.Equal(t, "Test Brand 1", got[0].CompanyName)
				assert.Equal(t, uint64(1000000), got[0].VolumeAverage)
				assert.Equal(t, "1002", got[1].TickerSymbol)
				assert.Equal(t, "Test Brand 2", got[1].CompanyName)
				assert.Equal(t, uint64(2000000), got[1].VolumeAverage)
			}
		})
	}
}
func TestHighVolumeStockBrandRepositoryImpl_convertToDomainModelWithName(t *testing.T) {
	now := time.Now()

	type args struct {
		dbModel     *genModel.HighVolumeStockBrand
		companyName string
	}
	tests := []struct {
		name string
		args args
		want *models.HighVolumeStockBrand
	}{
		{
			name: "正常系: DBモデルからドメインモデルへの変換",
			args: args{
				dbModel: &genModel.HighVolumeStockBrand{
					StockBrandID:  "uuid-1",
					TickerSymbol:  "1001",
					VolumeAverage: 1000000,
					CreatedAt:     now,
				},
				companyName: "Test Company",
			},
			want: models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Company", 1000000, now),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HighVolumeStockBrandRepositoryImpl{}
			got := r.convertToDomainModelWithName(tt.args.dbModel, tt.args.companyName)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToDomainModelWithName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

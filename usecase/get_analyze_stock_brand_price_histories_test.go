package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestStockBrandInteractorImpl_GetAnalyzeStockBrandPriceHistories(t *testing.T) {
	now := time.Now()
	sampleHistory := &models.AnalyzeStockBrandPriceHistory{
		ID:           "id-1",
		StockBrandID: "brand-1",
		TickerSymbol: "1234",
		TradePrice:   decimal.NewFromFloat(1000),
		CurrentPrice: decimal.NewFromFloat(1200),
		Action:       models.AnalyzeStockBrandPriceHistoryActionBuy,
		Method:       models.AnalyzeStockBrandPriceHistoryMethodSector25,
		CreatedAt:    now,
	}

	type fields struct {
		analyzeRepo func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository
	}
	type args struct {
		filter *models.AnalyzeStockBrandPriceHistoryFilter
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.PaginatedAnalyzeStockBrandPriceHistories
		wantErr bool
	}{
		{
			name: "正常系: page=1, limit=10, total=1",
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					expectedFilter := &models.AnalyzeStockBrandPriceHistoryFilter{
						Page:  1,
						Limit: 10,
					}
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(int64(1), nil)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return([]*models.AnalyzeStockBrandPriceHistory{sampleHistory}, nil)
					return m
				},
			},
			args: args{
				filter: &models.AnalyzeStockBrandPriceHistoryFilter{
					Page:  1,
					Limit: 10,
				},
			},
			want: &models.PaginatedAnalyzeStockBrandPriceHistories{
				Histories:  []*models.AnalyzeStockBrandPriceHistory{sampleHistory},
				Page:       1,
				Limit:      10,
				Total:      1,
				TotalPages: 1,
			},
		},
		{
			name: "正常系: total_pages の端数切り上げ",
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					expectedFilter := &models.AnalyzeStockBrandPriceHistoryFilter{
						Page:  2,
						Limit: 10,
					}
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(int64(25), nil)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return([]*models.AnalyzeStockBrandPriceHistory{sampleHistory}, nil)
					return m
				},
			},
			args: args{
				filter: &models.AnalyzeStockBrandPriceHistoryFilter{
					Page:  2,
					Limit: 10,
				},
			},
			want: &models.PaginatedAnalyzeStockBrandPriceHistories{
				Histories:  []*models.AnalyzeStockBrandPriceHistory{sampleHistory},
				Page:       2,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
			},
		},
		{
			name: "正常系: nilフィルタはデフォルト値を使う",
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					expectedFilter := &models.AnalyzeStockBrandPriceHistoryFilter{
						Page:  1,
						Limit: 100,
					}
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(int64(0), nil)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return([]*models.AnalyzeStockBrandPriceHistory{}, nil)
					return m
				},
			},
			args: args{filter: nil},
			want: &models.PaginatedAnalyzeStockBrandPriceHistories{
				Histories:  []*models.AnalyzeStockBrandPriceHistory{},
				Page:       1,
				Limit:      100,
				Total:      0,
				TotalPages: 1,
			},
		},
		{
			name: "異常系: CountWithFilter エラー",
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					expectedFilter := &models.AnalyzeStockBrandPriceHistoryFilter{
						Page:  1,
						Limit: 10,
					}
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(int64(0), assert.AnError)
					return m
				},
			},
			args: args{
				filter: &models.AnalyzeStockBrandPriceHistoryFilter{
					Page:  1,
					Limit: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "異常系: FindWithFilter エラー",
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					expectedFilter := &models.AnalyzeStockBrandPriceHistoryFilter{
						Page:  1,
						Limit: 10,
					}
					m.EXPECT().CountWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(int64(5), nil)
					m.EXPECT().FindWithFilter(gomock.Any(), gomock.Eq(expectedFilter)).Return(nil, assert.AnError)
					return m
				},
			},
			args: args{
				filter: &models.AnalyzeStockBrandPriceHistoryFilter{
					Page:  1,
					Limit: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			si := &stockBrandInteractorImpl{}
			if tt.fields.analyzeRepo != nil {
				si.analyzeStockBrandPriceHistoryRepository = tt.fields.analyzeRepo(ctrl)
			}

			got, err := si.GetAnalyzeStockBrandPriceHistories(context.Background(), tt.args.filter)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

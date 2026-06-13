package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func TestSectorPerformanceInteractor_GetSectorPerformance(t *testing.T) {
	from, _ := time.Parse("2006-01-02", "2024-01-01")
	to, _ := time.Parse("2006-01-02", "2024-03-31")

	sampleRows33 := []*models.Sector33AverageDailyPrice{
		{
			Date:       from,
			SectorCode: "3700",
			Adjclose:   decimal.NewFromFloat(1000.0),
		},
		{
			Date:       to,
			SectorCode: "3700",
			Adjclose:   decimal.NewFromFloat(1100.0),
		},
	}

	sampleRows17 := []*models.Sector17AverageDailyPrice{
		{
			Date:       from,
			SectorCode: "6",
			Adjclose:   decimal.NewFromFloat(2000.0),
		},
		{
			Date:       to,
			SectorCode: "6",
			Adjclose:   decimal.NewFromFloat(2200.0),
		},
	}

	type fields struct {
		sector33Repo func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository
		sector17Repo func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository
	}

	tests := []struct {
		name        string
		fields      fields
		from        time.Time
		to          time.Time
		granularity string
		want        func(result *models.SectorPerformance)
		wantErr     bool
	}{
		{
			name: "正常系（granularity=33）: データあり → SectorPerformance を返す",
			fields: fields{
				sector33Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(sampleRows33, nil)
					return m
				},
				sector17Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository {
					return mock_repositories.NewMockSector17AverageDailyPriceRepository(ctrl)
				},
			},
			from:        from,
			to:          to,
			granularity: "33",
			want: func(result *models.SectorPerformance) {
				assert.NotNil(t, result)
				assert.Equal(t, "33", result.Granularity)
				assert.Equal(t, "2024-01-01", result.From)
				assert.Equal(t, "2024-03-31", result.To)
				assert.Len(t, result.Sectors, 1)
				assert.Equal(t, "3700", result.Sectors[0].SectorCode)
				assert.Equal(t, "輸送用機器", result.Sectors[0].SectorName)
				assert.NotNil(t, result.Sectors[0].PeriodReturn)
			},
			wantErr: false,
		},
		{
			name: "正常系（granularity=17）: 17業種リポジトリが呼ばれる",
			fields: fields{
				sector33Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					return mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
				},
				sector17Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector17AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(sampleRows17, nil)
					return m
				},
			},
			from:        from,
			to:          to,
			granularity: "17",
			want: func(result *models.SectorPerformance) {
				assert.NotNil(t, result)
				assert.Equal(t, "17", result.Granularity)
				assert.Equal(t, "2024-01-01", result.From)
				assert.Equal(t, "2024-03-31", result.To)
				assert.Len(t, result.Sectors, 1)
				assert.Equal(t, "6", result.Sectors[0].SectorCode)
				assert.Equal(t, "自動車・輸送機", result.Sectors[0].SectorName)
				assert.NotNil(t, result.Sectors[0].PeriodReturn)
			},
			wantErr: false,
		},
		{
			name: "正常系（granularity=33 デフォルト）: データなし → Sectors が空",
			fields: fields{
				sector33Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, nil)
					return m
				},
				sector17Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository {
					return mock_repositories.NewMockSector17AverageDailyPriceRepository(ctrl)
				},
			},
			from:        from,
			to:          to,
			granularity: "33",
			want: func(result *models.SectorPerformance) {
				assert.NotNil(t, result)
				assert.Empty(t, result.Sectors)
			},
			wantErr: false,
		},
		{
			name: "異常系（sector33）: リポジトリエラー → エラーを返す",
			fields: fields{
				sector33Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db error"))
					return m
				},
				sector17Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository {
					return mock_repositories.NewMockSector17AverageDailyPriceRepository(ctrl)
				},
			},
			from:        from,
			to:          to,
			granularity: "33",
			wantErr:     true,
		},
		{
			name: "異常系（sector17）: リポジトリエラー → エラーを返す",
			fields: fields{
				sector33Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					return mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
				},
				sector17Repo: func(ctrl *gomock.Controller) *mock_repositories.MockSector17AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector17AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db error"))
					return m
				},
			},
			from:        from,
			to:          to,
			granularity: "17",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := &sectorPerformanceInteractorImpl{
				sector33Repo: tt.fields.sector33Repo(ctrl),
				sector17Repo: tt.fields.sector17Repo(ctrl),
			}

			got, err := s.GetSectorPerformance(context.Background(), tt.from, tt.to, tt.granularity)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSectorPerformance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				tt.want(got)
			}
		})
	}
}

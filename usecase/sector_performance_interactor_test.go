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

	sampleRows := []*models.Sector33AverageDailyPrice{
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

	type fields struct {
		sectorRepo func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository
	}

	tests := []struct {
		name    string
		fields  fields
		from    time.Time
		to      time.Time
		want    func(result *models.SectorPerformance)
		wantErr bool
	}{
		{
			name: "正常系: データあり → SectorPerformance を返す",
			fields: fields{
				sectorRepo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(sampleRows, nil)
					return m
				},
			},
			from: from,
			to:   to,
			want: func(result *models.SectorPerformance) {
				assert.NotNil(t, result)
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
			name: "正常系: データなし → Sectors が空",
			fields: fields{
				sectorRepo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, nil)
					return m
				},
			},
			from: from,
			to:   to,
			want: func(result *models.SectorPerformance) {
				assert.NotNil(t, result)
				assert.Empty(t, result.Sectors)
			},
			wantErr: false,
		},
		{
			name: "異常系: リポジトリエラー → エラーを返す",
			fields: fields{
				sectorRepo: func(ctrl *gomock.Controller) *mock_repositories.MockSector33AverageDailyPriceRepository {
					m := mock_repositories.NewMockSector33AverageDailyPriceRepository(ctrl)
					m.EXPECT().ListRangeAll(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db error"))
					return m
				},
			},
			from:    from,
			to:      to,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := &sectorPerformanceInteractorImpl{
				sectorRepo: tt.fields.sectorRepo(ctrl),
			}

			got, err := s.GetSectorPerformance(context.Background(), tt.from, tt.to)
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

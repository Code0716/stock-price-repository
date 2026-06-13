//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

type sectorPerformanceInteractorImpl struct {
	sector33Repo repositories.Sector33AverageDailyPriceRepository
	sector17Repo repositories.Sector17AverageDailyPriceRepository
}

// SectorPerformanceInteractor セクターパフォーマンス API のユースケース
type SectorPerformanceInteractor interface {
	// GetSectorPerformance 指定期間の業種別パフォーマンスを算出する。
	// granularity は "33"（デフォルト）または "17" を受け取る。
	GetSectorPerformance(ctx context.Context, from, to time.Time, granularity string) (*models.SectorPerformance, error)
}

// NewSectorPerformanceInteractor コンストラクタ
func NewSectorPerformanceInteractor(
	sector33Repo repositories.Sector33AverageDailyPriceRepository,
	sector17Repo repositories.Sector17AverageDailyPriceRepository,
) SectorPerformanceInteractor {
	return &sectorPerformanceInteractorImpl{
		sector33Repo: sector33Repo,
		sector17Repo: sector17Repo,
	}
}

func (s *sectorPerformanceInteractorImpl) GetSectorPerformance(ctx context.Context, from, to time.Time, granularity string) (*models.SectorPerformance, error) {
	var items []*models.SectorPerformanceItem

	switch granularity {
	case "17":
		rows, err := s.sector17Repo.ListRangeAll(ctx, from, to)
		if err != nil {
			return nil, errors.Wrap(err, "sectorPerformanceInteractorImpl.GetSectorPerformance: sector17 ListRangeAll")
		}
		items = domain_service.CalcSector17Performance(rows, models.Sector17Codes)
	default:
		// "33" またはデフォルト
		rows, err := s.sector33Repo.ListRangeAll(ctx, from, to)
		if err != nil {
			return nil, errors.Wrap(err, "sectorPerformanceInteractorImpl.GetSectorPerformance: sector33 ListRangeAll")
		}
		items = domain_service.CalcSectorPerformance(rows, models.Sector33Codes)
	}

	return &models.SectorPerformance{
		Granularity: granularity,
		From:        from.Format(util.DateLayout),
		To:          to.Format(util.DateLayout),
		Sectors:     items,
	}, nil
}

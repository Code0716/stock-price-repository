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
	sectorRepo repositories.Sector33AverageDailyPriceRepository
}

// SectorPerformanceInteractor セクターパフォーマンス API のユースケース
type SectorPerformanceInteractor interface {
	// GetSectorPerformance 指定期間の業種別パフォーマンスを算出する。
	GetSectorPerformance(ctx context.Context, from, to time.Time) (*models.SectorPerformance, error)
}

// NewSectorPerformanceInteractor コンストラクタ
func NewSectorPerformanceInteractor(
	sectorRepo repositories.Sector33AverageDailyPriceRepository,
) SectorPerformanceInteractor {
	return &sectorPerformanceInteractorImpl{
		sectorRepo: sectorRepo,
	}
}

func (s *sectorPerformanceInteractorImpl) GetSectorPerformance(ctx context.Context, from, to time.Time) (*models.SectorPerformance, error) {
	rows, err := s.sectorRepo.ListRangeAll(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "sectorPerformanceInteractorImpl.GetSectorPerformance: ListRangeAll")
	}

	// 業種名マップは models.Sector33Codes をそのまま利用
	items := domain_service.CalcSectorPerformance(rows, models.Sector33Codes)

	return &models.SectorPerformance{
		From:    from.Format(util.DateLayout),
		To:      to.Format(util.DateLayout),
		Sectors: items,
	}, nil
}

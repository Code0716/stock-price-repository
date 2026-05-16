package usecase

import (
	"context"
	"math"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
)

// GetFinAnnouncements 決算発表予定一覧を取得する
func (si *stockBrandInteractorImpl) GetFinAnnouncements(ctx context.Context, filter *models.FinAnnouncementFilter) (*models.PaginatedFinAnnouncements, error) {
	if filter == nil {
		filter = &models.FinAnnouncementFilter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	total, err := si.finAnnouncementRepository.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "finAnnouncementRepository.CountWithFilter error")
	}

	announcements, err := si.finAnnouncementRepository.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "finAnnouncementRepository.FindWithFilter error")
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &models.PaginatedFinAnnouncements{
		Announcements: announcements,
		Page:          filter.Page,
		Limit:         filter.Limit,
		Total:         total,
		TotalPages:    totalPages,
	}, nil
}

// GetNextFinAnnouncement 銘柄の次回決算発表予定を取得する
func (si *stockBrandInteractorImpl) GetNextFinAnnouncement(ctx context.Context, tickerSymbol string) (*models.FinAnnouncement, error) {
	result, err := si.finAnnouncementRepository.FindNextBySymbol(ctx, tickerSymbol)
	if err != nil {
		return nil, errors.Wrap(err, "finAnnouncementRepository.FindNextBySymbol error")
	}
	return result, nil
}

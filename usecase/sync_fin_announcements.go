package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
)

// SyncFinAnnouncements j-Quantsから決算発表予定を取得してDBに保存する
func (si *stockBrandInteractorImpl) SyncFinAnnouncements(ctx context.Context) error {
	infos, err := si.stockAPIClient.GetAnnounceFinSchedule(ctx)
	if err != nil {
		return errors.Wrap(err, "GetAnnounceFinSchedule error")
	}

	now := time.Now()
	announcements := make([]*models.FinAnnouncement, 0, len(infos))
	for _, info := range infos {
		announcements = append(announcements, &models.FinAnnouncement{
			ID:               uuid.NewString(),
			TickerSymbol:     info.Code,
			AnnouncementDate: info.Date,
			FiscalYear:       info.FiscalYear,
			FiscalQuarter:    info.FiscalQuarter,
			Sector17Code:     "",
			Sector33Code:     "",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	if err := si.finAnnouncementRepository.Upsert(ctx, announcements); err != nil {
		return errors.Wrap(err, "finAnnouncementRepository.Upsert error")
	}
	return nil
}

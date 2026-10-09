//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// MarketCalendar 営業日・休場日と市場イベントのセット。
type MarketCalendar struct {
	Days   []*models.MarketCalendarDay
	Events []*models.MarketEvent
}

type MarketCalendarInteractor interface {
	// GetMarketCalendar from〜to（両端含む）の取引カレンダーと市場イベントを取得する。
	GetMarketCalendar(ctx context.Context, from, to time.Time) (*MarketCalendar, error)
}

type marketCalendarInteractorImpl struct {
	marketCalendarRepository repositories.MarketCalendarRepository
}

func NewMarketCalendarInteractor(marketCalendarRepository repositories.MarketCalendarRepository) MarketCalendarInteractor {
	return &marketCalendarInteractorImpl{marketCalendarRepository: marketCalendarRepository}
}

func (mi *marketCalendarInteractorImpl) GetMarketCalendar(ctx context.Context, from, to time.Time) (*MarketCalendar, error) {
	days, err := mi.marketCalendarRepository.ListDays(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "ListDays error")
	}
	events, err := mi.marketCalendarRepository.ListEvents(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "ListEvents error")
	}
	return &MarketCalendar{Days: days, Events: events}, nil
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type MarketCalendarRepositoryImpl struct {
	query *genQuery.Query
}

func NewMarketCalendarRepositoryImpl(db *gorm.DB) repositories.MarketCalendarRepository {
	return &MarketCalendarRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (mi *MarketCalendarRepositoryImpl) UpsertDays(ctx context.Context, days []*models.MarketCalendarDay) error {
	tx := TxOrDefault(ctx, mi.query)

	if len(days) == 0 {
		return nil
	}

	rows := make([]*genModel.MarketCalendar, 0, len(days))
	for _, d := range days {
		rows = append(rows, &genModel.MarketCalendar{
			CalendarDate: dateOnlyOf(d.Date),
			HolDiv:       int32(d.HolDiv),
		})
	}

	if err := tx.MarketCalendar.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "calendar_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"hol_div", "updated_at"}),
		}).
		Create(rows...); err != nil {
		return errors.Wrap(err, "MarketCalendarRepositoryImpl.UpsertDays error")
	}
	return nil
}

func (mi *MarketCalendarRepositoryImpl) ListDays(ctx context.Context, from, to time.Time) ([]*models.MarketCalendarDay, error) {
	tx := TxOrDefault(ctx, mi.query)

	rows, err := tx.MarketCalendar.WithContext(ctx).
		Where(tx.MarketCalendar.CalendarDate.Gte(dateOnlyOf(from)), tx.MarketCalendar.CalendarDate.Lte(dateOnlyOf(to))).
		Order(tx.MarketCalendar.CalendarDate).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "MarketCalendarRepositoryImpl.ListDays error")
	}

	out := make([]*models.MarketCalendarDay, 0, len(rows))
	for _, r := range rows {
		out = append(out, &models.MarketCalendarDay{Date: r.CalendarDate, HolDiv: int(r.HolDiv)})
	}
	return out, nil
}

func (mi *MarketCalendarRepositoryImpl) ReplaceEvents(ctx context.Context, kind string, from, to time.Time, events []*models.MarketEvent) error {
	tx := TxOrDefault(ctx, mi.query)

	if _, err := tx.MarketEvent.WithContext(ctx).
		Where(
			tx.MarketEvent.Kind.Eq(kind),
			tx.MarketEvent.EventDate.Gte(dateOnlyOf(from)),
			tx.MarketEvent.EventDate.Lte(dateOnlyOf(to)),
		).
		Delete(); err != nil {
		return errors.Wrap(err, "MarketCalendarRepositoryImpl.ReplaceEvents delete error")
	}

	if len(events) == 0 {
		return nil
	}

	rows := make([]*genModel.MarketEvent, 0, len(events))
	for _, e := range events {
		rows = append(rows, &genModel.MarketEvent{
			EventDate: dateOnlyOf(e.Date),
			Kind:      e.Kind,
			Label:     e.Label,
			Source:    e.Source,
		})
	}
	if err := tx.MarketEvent.WithContext(ctx).Create(rows...); err != nil {
		return errors.Wrap(err, "MarketCalendarRepositoryImpl.ReplaceEvents create error")
	}
	return nil
}

func (mi *MarketCalendarRepositoryImpl) ListEvents(ctx context.Context, from, to time.Time) ([]*models.MarketEvent, error) {
	tx := TxOrDefault(ctx, mi.query)

	rows, err := tx.MarketEvent.WithContext(ctx).
		Where(tx.MarketEvent.EventDate.Gte(dateOnlyOf(from)), tx.MarketEvent.EventDate.Lte(dateOnlyOf(to))).
		Order(tx.MarketEvent.EventDate, tx.MarketEvent.Kind).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "MarketCalendarRepositoryImpl.ListEvents error")
	}

	out := make([]*models.MarketEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, &models.MarketEvent{Date: r.EventDate, Kind: r.Kind, Label: r.Label, Source: r.Source})
	}
	return out, nil
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type DailyAvoidStockRepositoryImpl struct {
	query *genQuery.Query
}

func NewDailyAvoidStockRepositoryImpl(db *gorm.DB) repositories.DailyAvoidStockRepository {
	return &DailyAvoidStockRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (di *DailyAvoidStockRepositoryImpl) BulkCreate(ctx context.Context, rows []*models.DailyAvoidStock) error {
	tx := TxOrDefault(ctx, di.query)

	if len(rows) == 0 {
		return nil
	}

	if err := tx.DailyAvoidStock.WithContext(ctx).
		Create(di.convertToDBModels(rows)...); err != nil {
		return errors.Wrap(err, "DailyAvoidStockRepositoryImpl.BulkCreate error")
	}
	return nil
}

func (di *DailyAvoidStockRepositoryImpl) DeleteByAsOfDate(ctx context.Context, asOfDate time.Time) error {
	tx := TxOrDefault(ctx, di.query)

	if _, err := tx.DailyAvoidStock.WithContext(ctx).
		Where(tx.DailyAvoidStock.AsOfDate.Eq(dateOnlyOf(asOfDate))).
		Delete(); err != nil {
		return errors.Wrap(err, "DailyAvoidStockRepositoryImpl.DeleteByAsOfDate error")
	}
	return nil
}

func (di *DailyAvoidStockRepositoryImpl) ListByAsOfDate(ctx context.Context, asOfDate time.Time) ([]*models.DailyAvoidStock, error) {
	tx := TxOrDefault(ctx, di.query)

	rows, err := tx.DailyAvoidStock.WithContext(ctx).
		Where(tx.DailyAvoidStock.AsOfDate.Eq(dateOnlyOf(asOfDate))).
		Order(tx.DailyAvoidStock.AvoidRank).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "DailyAvoidStockRepositoryImpl.ListByAsOfDate error")
	}

	out := make([]*models.DailyAvoidStock, 0, len(rows))
	for _, r := range rows {
		out = append(out, di.convertToDomainModel(r))
	}
	return out, nil
}

func (di *DailyAvoidStockRepositoryImpl) ExistsByAsOfDate(ctx context.Context, asOfDate time.Time) (bool, error) {
	tx := TxOrDefault(ctx, di.query)

	count, err := tx.DailyAvoidStock.WithContext(ctx).
		Where(tx.DailyAvoidStock.AsOfDate.Eq(dateOnlyOf(asOfDate))).
		Count()
	if err != nil {
		return false, errors.Wrap(err, "DailyAvoidStockRepositoryImpl.ExistsByAsOfDate error")
	}
	return count > 0, nil
}

func (di *DailyAvoidStockRepositoryImpl) FindLatestAsOfDate(ctx context.Context) (*time.Time, error) {
	tx := TxOrDefault(ctx, di.query)

	row, err := tx.DailyAvoidStock.WithContext(ctx).
		Order(tx.DailyAvoidStock.AsOfDate.Desc()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "DailyAvoidStockRepositoryImpl.FindLatestAsOfDate error")
	}

	return &row.AsOfDate, nil
}

func (di *DailyAvoidStockRepositoryImpl) ListAsOfDates(ctx context.Context, limit int) ([]time.Time, error) {
	tx := TxOrDefault(ctx, di.query)

	q := tx.DailyAvoidStock.WithContext(ctx).
		Distinct(tx.DailyAvoidStock.AsOfDate).
		Order(tx.DailyAvoidStock.AsOfDate.Desc())
	if limit > 0 {
		q = q.Limit(limit)
	}

	var dates []time.Time
	if err := q.Pluck(tx.DailyAvoidStock.AsOfDate, &dates); err != nil {
		return nil, errors.Wrap(err, "DailyAvoidStockRepositoryImpl.ListAsOfDates error")
	}
	return dates, nil
}

func (di *DailyAvoidStockRepositoryImpl) convertToDomainModel(m *genModel.DailyAvoidStock) *models.DailyAvoidStock {
	out := &models.DailyAvoidStock{
		AsOfDate:             m.AsOfDate,
		StockBrandID:         m.StockBrandID,
		TickerSymbol:         m.TickerSymbol,
		AvoidRank:            int(m.AvoidRank),
		Severity:             models.DailyAvoidSeverity(m.Severity),
		Reason:               m.Reason,
		RuleVersion:          m.RuleVersion,
		Volatility12M:        decimal.NewFromFloat(m.Volatility12M),
		VolatilityPercentile: decimal.NewFromFloat(m.VolatilityPercentile),
		UniverseSize:         int(m.UniverseSize),
		ThresholdVolatility:  decimal.NewFromFloat(m.ThresholdVolatility),
		AvgTradingValue:      decimal.NewFromFloat(m.AvgTradingValue),
		BaseClosePrice:       decimal.NewFromFloat(m.BaseClosePrice),
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
	if m.Sector33CodeName != nil {
		out.Sector33CodeName = *m.Sector33CodeName
	}
	return out
}

func (di *DailyAvoidStockRepositoryImpl) convertToDBModel(a *models.DailyAvoidStock) *genModel.DailyAvoidStock {
	m := &genModel.DailyAvoidStock{
		AsOfDate:             dateOnlyOf(a.AsOfDate),
		StockBrandID:         a.StockBrandID,
		TickerSymbol:         a.TickerSymbol,
		AvoidRank:            uint32(a.AvoidRank),
		Severity:             string(a.Severity),
		Reason:               a.Reason,
		RuleVersion:          a.RuleVersion,
		Volatility12M:        roundToFloat64(a.Volatility12M, 6),
		VolatilityPercentile: roundToFloat64(a.VolatilityPercentile, 4),
		UniverseSize:         uint32(a.UniverseSize),
		ThresholdVolatility:  roundToFloat64(a.ThresholdVolatility, 6),
		AvgTradingValue:      roundToFloat64(a.AvgTradingValue, 4),
		BaseClosePrice:       roundToFloat64(a.BaseClosePrice, 4),
	}
	if a.Sector33CodeName != "" {
		v := a.Sector33CodeName
		m.Sector33CodeName = &v
	}
	return m
}

func (di *DailyAvoidStockRepositoryImpl) convertToDBModels(rows []*models.DailyAvoidStock) []*genModel.DailyAvoidStock {
	out := make([]*genModel.DailyAvoidStock, 0, len(rows))
	for _, a := range rows {
		out = append(out, di.convertToDBModel(a))
	}
	return out
}

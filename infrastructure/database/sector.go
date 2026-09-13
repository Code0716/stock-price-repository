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

// --- Sector33 ---

// Sector33AverageDailyPriceRepositoryImpl implements Sector33AverageDailyPriceRepository
type Sector33AverageDailyPriceRepositoryImpl struct {
	query *genQuery.Query
}

func NewSector33AverageDailyPriceRepositoryImpl(db *gorm.DB) repositories.Sector33AverageDailyPriceRepository {
	return &Sector33AverageDailyPriceRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (r *Sector33AverageDailyPriceRepositoryImpl) ListRangeAll(ctx context.Context, from, to time.Time) ([]*models.Sector33AverageDailyPrice, error) {
	tx := TxOrDefault(ctx, r.query)

	q := tx.Sector33AverageDailyPrice.WithContext(ctx)

	dateFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	dateTo := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

	// NULL の sector_33_code を除外
	q = q.Where(tx.Sector33AverageDailyPrice.Sector33Code.IsNotNull())
	q = q.Where(tx.Sector33AverageDailyPrice.Date.Gte(dateFrom))
	q = q.Where(tx.Sector33AverageDailyPrice.Date.Lte(dateTo))

	rows, err := q.Order(tx.Sector33AverageDailyPrice.Date).Find()
	if err != nil {
		return nil, errors.Wrap(err, "Sector33AverageDailyPriceRepositoryImpl.ListRangeAll error")
	}

	result := make([]*models.Sector33AverageDailyPrice, 0, len(rows))
	for _, row := range rows {
		result = append(result, r.convertToDomainModel(row))
	}
	return result, nil
}

func (r *Sector33AverageDailyPriceRepositoryImpl) convertToDomainModel(m *genModel.Sector33AverageDailyPrice) *models.Sector33AverageDailyPrice {
	if m == nil {
		return nil
	}
	sectorCode := ""
	if m.Sector33Code != nil {
		sectorCode = *m.Sector33Code
	}
	return &models.Sector33AverageDailyPrice{
		Date:       m.Date,
		SectorCode: sectorCode,
		Open:       decimal.NewFromFloat(m.OpenPrice),
		Close:      decimal.NewFromFloat(m.ClosePrice),
		High:       decimal.NewFromFloat(m.HighPrice),
		Low:        decimal.NewFromFloat(m.LowPrice),
		Adjclose:   decimal.NewFromFloat(m.AdjClosePrice),
	}
}

// --- Sector17 ---

// Sector17AverageDailyPriceRepositoryImpl implements Sector17AverageDailyPriceRepository
type Sector17AverageDailyPriceRepositoryImpl struct {
	query *genQuery.Query
}

func NewSector17AverageDailyPriceRepositoryImpl(db *gorm.DB) repositories.Sector17AverageDailyPriceRepository {
	return &Sector17AverageDailyPriceRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (r *Sector17AverageDailyPriceRepositoryImpl) ListRangeAll(ctx context.Context, from, to time.Time) ([]*models.Sector17AverageDailyPrice, error) {
	tx := TxOrDefault(ctx, r.query)

	q := tx.Sector17AverageDailyPrice.WithContext(ctx)

	dateFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	dateTo := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

	// NULL の sector_17_code を除外
	q = q.Where(tx.Sector17AverageDailyPrice.Sector17Code.IsNotNull())
	q = q.Where(tx.Sector17AverageDailyPrice.Date.Gte(dateFrom))
	q = q.Where(tx.Sector17AverageDailyPrice.Date.Lte(dateTo))

	rows, err := q.Order(tx.Sector17AverageDailyPrice.Date).Find()
	if err != nil {
		return nil, errors.Wrap(err, "Sector17AverageDailyPriceRepositoryImpl.ListRangeAll error")
	}

	result := make([]*models.Sector17AverageDailyPrice, 0, len(rows))
	for _, row := range rows {
		result = append(result, r.convertToDomainModel17(row))
	}
	return result, nil
}

func (r *Sector17AverageDailyPriceRepositoryImpl) convertToDomainModel17(m *genModel.Sector17AverageDailyPrice) *models.Sector17AverageDailyPrice {
	if m == nil {
		return nil
	}
	sectorCode := ""
	if m.Sector17Code != nil {
		sectorCode = *m.Sector17Code
	}
	return &models.Sector17AverageDailyPrice{
		Date:       m.Date,
		SectorCode: sectorCode,
		Open:       decimal.NewFromFloat(m.OpenPrice),
		Close:      decimal.NewFromFloat(m.ClosePrice),
		High:       decimal.NewFromFloat(m.HighPrice),
		Low:        decimal.NewFromFloat(m.LowPrice),
		Adjclose:   decimal.NewFromFloat(m.AdjClosePrice),
	}
}

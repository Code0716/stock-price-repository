//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// TopixRepositoryImpl implements TopixRepository
type TopixRepositoryImpl struct {
	query *genQuery.Query
}

func NewTopixRepositoryImpl(db *gorm.DB) repositories.TopixRepository {
	return &TopixRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (tr *TopixRepositoryImpl) CreateTopixDailyPrices(ctx context.Context, dailyPrices models.IndexStockAverageDailyPrices) error {
	query := TxOrDefault(ctx, tr.query)

	if err := query.
		TopixDailyPrice.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "date"}},
			DoUpdates: clause.AssignmentColumns(
				[]string{
					"open_price",
					"close_price",
					"high_price",
					"low_price",
					"adj_close_price",
					"updated_at",
				}),
		}).
		Create(tr.topixDailyPricesToGenModel(dailyPrices)...); err != nil {
		return errors.Wrap(err, "CreateTopixDailyPrices error")
	}
	return nil
}

func (tr *TopixRepositoryImpl) ListTopixDailyPrices(ctx context.Context, from, to *time.Time) (models.IndexStockAverageDailyPrices, error) {
	tx := TxOrDefault(ctx, tr.query)

	do := tx.TopixDailyPrice.WithContext(ctx)
	if from != nil {
		dateFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		do = do.Where(tx.TopixDailyPrice.Date.Gte(dateFrom))
	}
	if to != nil {
		dateTo := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())
		do = do.Where(tx.TopixDailyPrice.Date.Lte(dateTo))
	}

	rows, err := do.Order(tx.TopixDailyPrice.Date).Find()
	if err != nil {
		return nil, errors.Wrap(err, "ListTopixDailyPrices error")
	}

	result := make(models.IndexStockAverageDailyPrices, 0, len(rows))
	for _, row := range rows {
		result = append(result, tr.convertToDomainModel(row))
	}
	return result, nil
}

func (tr *TopixRepositoryImpl) convertToDomainModel(m *genModel.TopixDailyPrice) *models.IndexStockAverageDailyPrice {
	if m == nil {
		return nil
	}
	return &models.IndexStockAverageDailyPrice{
		Date:      m.Date,
		Open:      decimal.NewFromFloat(m.OpenPrice),
		Close:     decimal.NewFromFloat(m.ClosePrice),
		High:      decimal.NewFromFloat(m.HighPrice),
		Low:       decimal.NewFromFloat(m.LowPrice),
		Adjclose:  decimal.NewFromFloat(m.AdjClosePrice),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (tr *TopixRepositoryImpl) topixDailyPricesToGenModel(dailyPrices models.IndexStockAverageDailyPrices) []*genModel.TopixDailyPrice {
	var result []*genModel.TopixDailyPrice
	for _, v := range dailyPrices {
		open, _ := v.Open.Round(4).Float64()
		closePrice, _ := v.Close.Round(4).Float64()
		high, _ := v.High.Round(4).Float64()
		low, _ := v.Low.Round(4).Float64()
		adjclose, _ := v.Adjclose.Round(4).Float64()

		result = append(result, &genModel.TopixDailyPrice{
			Date:          v.Date,
			OpenPrice:     open,
			ClosePrice:    closePrice,
			HighPrice:     high,
			LowPrice:      low,
			AdjClosePrice: adjclose,
			CreatedAt:     v.CreatedAt,
			UpdatedAt:     v.UpdatedAt,
		})
	}
	return result
}

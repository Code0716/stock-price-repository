//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE

package database

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// DjiRepositoryImpl implements  DjiRepository
type DjiRepositoryImpl struct {
	query *genQuery.Query
}

func NewDjiRepositoryImpl(db *gorm.DB) repositories.DjiRepository {
	return &DjiRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (ni *DjiRepositoryImpl) CreateDjiStockAverageDailyPrices(ctx context.Context, averageDailyPrices models.IndexStockAverageDailyPrices) error {
	query, ok := GetTxQuery(ctx)
	if !ok {
		query = ni.query
	}

	if err := query.DjiStockAverageDailyStockPrice.WithContext(ctx).
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
		Create(ni.DjiStockAverageDailyPricesToGenModel(averageDailyPrices)...); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (ni *DjiRepositoryImpl) DjiStockAverageDailyPricesToGenModel(averageDailyPrices models.IndexStockAverageDailyPrices) []*genModel.DjiStockAverageDailyStockPrice {
	var averageDailyPricesGne []*genModel.DjiStockAverageDailyStockPrice
	for _, v := range averageDailyPrices {
		open, _ := v.Open.Round(4).Float64()
		closePrice, _ := v.Close.Round(4).Float64()
		high, _ := v.High.Round(4).Float64()
		low, _ := v.Low.Round(4).Float64()
		Adjclose, _ := v.Adjclose.Round(4).Float64()
		averageDailyPricesGne = append(averageDailyPricesGne, &genModel.DjiStockAverageDailyStockPrice{
			Date:          v.Date,
			OpenPrice:     open,
			ClosePrice:    closePrice,
			HighPrice:     high,
			LowPrice:      low,
			AdjClosePrice: Adjclose,
			CreatedAt:     v.CreatedAt,
			UpdatedAt:     v.UpdatedAt,
		})
	}

	return averageDailyPricesGne
}

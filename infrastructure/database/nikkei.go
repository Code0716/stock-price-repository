//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NikkeiRepositoryImpl implements  NikkeiRepository
type NikkeiRepositoryImpl struct {
	query *genQuery.Query
}

func NewNikkeiRepositoryImpl(db *gorm.DB) repositories.NikkeiRepository {
	return &NikkeiRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (ni *NikkeiRepositoryImpl) CreateNikkeiStockAverageDailyPrices(ctx context.Context, averageDailyPrices models.IndexStockAverageDailyPrices) error {
	query, ok := GetTxQuery(ctx)
	if !ok {
		query = ni.query
	}

	if err := query.
		NikkeiStockAverageDailyPrice.WithContext(ctx).
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
		Create(ni.nikkeiStockAverageDailyPricesToGenModel(averageDailyPrices)...); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (ni *NikkeiRepositoryImpl) nikkeiStockAverageDailyPricesToGenModel(averageDailyPrices models.IndexStockAverageDailyPrices) []*genModel.NikkeiStockAverageDailyPrice {
	var averageDailyPricesGne []*genModel.NikkeiStockAverageDailyPrice
	for _, v := range averageDailyPrices {
		open, _ := v.Open.Round(4).Float64()
		close, _ := v.Close.Round(4).Float64()
		high, _ := v.High.Round(4).Float64()
		low, _ := v.Low.Round(4).Float64()
		adjclose, _ := v.Adjclose.Round(4).Float64()

		// DBをdatetimeにしてしまっているので、時間を9時に。
		// 本来良くないが、一旦こちらで対応する。
		newDatetime := time.Date(v.Date.Year(), v.Date.Month(), v.Date.Day(), 9, 0, 0, 0, v.Date.Location())
		averageDailyPricesGne = append(averageDailyPricesGne, &genModel.NikkeiStockAverageDailyPrice{
			Date:          newDatetime,
			OpenPrice:     open,
			ClosePrice:    close,
			HighPrice:     high,
			LowPrice:      low,
			AdjClosePrice: adjclose,
			CreatedAt:     v.CreatedAt,
			UpdatedAt:     v.UpdatedAt,
		})
	}

	return averageDailyPricesGne
}

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

type StockBrandsDailyPriceRepositoryImpl struct {
	query *genQuery.Query
}

func NewStockBrandsDailyPriceRepositoryImpl(db *gorm.DB) repositories.StockBrandsDailyPriceRepository {
	return &StockBrandsDailyPriceRepositoryImpl{
		query: genQuery.Use(db),
	}
}
func (si *StockBrandsDailyPriceRepositoryImpl) GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error) {

	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}
	price, err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Where(tx.StockBrandsDailyPrice.TickerSymbol.Eq(symbol)).
		Order(tx.StockBrandsDailyPrice.Date.Desc()).
		First()

	if err != nil {
		return nil, errors.Wrap(err, "StockBrandsDailyPrice.GetLatestPriceBySymbol error")
	}

	return si.convertToDomainModel(price), nil

}

func (si *StockBrandsDailyPriceRepositoryImpl) CreateStockBrandDailyPrice(ctx context.Context, dailyPrices []*models.StockBrandDailyPrice) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ticker_symbol"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns(
				[]string{
					"open_price",
					"close_price",
					"high_price",
					"low_price",
					"adj_close_price",
					"volume",
					"updated_at",
				}),
		}).
		Create(si.convertToDBModels(dailyPrices)...); err != nil {
		return errors.Wrap(err, "CreateStockBrandsDailyPrice error")
	}
	return nil
}

func (si *StockBrandsDailyPriceRepositoryImpl) DeleteByIDs(ctx context.Context, ids []string) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if _, err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Where(tx.StockBrandsDailyPrice.StockBrandID.In(ids...)).
		Delete(); err != nil {
		return errors.Wrap(err, "StockBrandsDailyPrice.DeleteDelisting error")
	}
	return nil
}

func (si *StockBrandsDailyPriceRepositoryImpl) ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if filter.TickerSymbol == "" {
		return nil, errors.New("TickerSymbol is required")
	}
	q := tx.WithContext(ctx).
		StockBrandsDailyPrice.
		Where(tx.StockBrandsDailyPrice.
			TickerSymbol.
			Eq(filter.TickerSymbol),
		)

	if filter.DateFrom != nil {
		dateOnlyFrom := time.Date(
			filter.DateFrom.Year(),
			filter.DateFrom.Month(),
			filter.DateFrom.Day(),
			0, 0, 0, 0,
			filter.DateFrom.Location(),
		)

		q = q.Where(tx.StockBrandsDailyPrice.Date.Gte(dateOnlyFrom))
	}

	if filter.DateTo != nil {
		dateOnlyTo := time.Date(
			filter.DateTo.Year(),
			filter.DateTo.Month(),
			filter.DateTo.Day(),
			0, 0, 0, 0,
			filter.DateTo.Location(),
		)
		q = q.Where(tx.StockBrandsDailyPrice.Date.Lte(dateOnlyTo))
	}

	rdbDailyPrices, err := q.Find()
	if err != nil {
		return nil, errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	domainDailyPrices := make([]*models.StockBrandDailyPrice, 0, len(rdbDailyPrices))
	for _, rdbDailyPrice := range rdbDailyPrices {
		domainDailyPrices = append(domainDailyPrices, si.convertToDomainModel(rdbDailyPrice))
	}

	return domainDailyPrices, nil
}

func (si *StockBrandsDailyPriceRepositoryImpl) convertToDomainModel(dailyPriceDB *genModel.StockBrandsDailyPrice) *models.StockBrandDailyPrice {
	if dailyPriceDB == nil {
		return nil
	}

	return &models.StockBrandDailyPrice{
		ID:           dailyPriceDB.ID,
		TickerSymbol: dailyPriceDB.TickerSymbol,
		Date:         dailyPriceDB.Date,
		Open:         decimal.NewFromFloat(dailyPriceDB.OpenPrice),
		Close:        decimal.NewFromFloat(dailyPriceDB.ClosePrice),
		High:         decimal.NewFromFloat(dailyPriceDB.HighPrice),
		Low:          decimal.NewFromFloat(dailyPriceDB.LowPrice),
		Adjclose:     decimal.NewFromFloat(dailyPriceDB.AdjClosePrice),
		Volume:       int64(dailyPriceDB.Volume),
		CreatedAt:    dailyPriceDB.CreatedAt,
		UpdatedAt:    dailyPriceDB.UpdatedAt,
	}
}

func (si *StockBrandsDailyPriceRepositoryImpl) convertToDBModels(dailyPrices []*models.StockBrandDailyPrice) []*genModel.StockBrandsDailyPrice {
	var dailyPricesDB []*genModel.StockBrandsDailyPrice
	for _, v := range dailyPrices {
		dailyPricesDB = append(dailyPricesDB, si.convertToDBModel(v))
	}
	return dailyPricesDB
}

func (si *StockBrandsDailyPriceRepositoryImpl) convertToDBModel(dailyPrice *models.StockBrandDailyPrice) *genModel.StockBrandsDailyPrice {
	open, _ := dailyPrice.Open.Round(4).Float64()
	closePrice, _ := dailyPrice.Close.Round(4).Float64()
	high, _ := dailyPrice.High.Round(4).Float64()
	low, _ := dailyPrice.Low.Round(4).Float64()
	adjclose, _ := dailyPrice.Adjclose.Round(4).Float64()
	return &genModel.StockBrandsDailyPrice{
		ID:            dailyPrice.ID,
		StockBrandID:  dailyPrice.StockBrandID,
		TickerSymbol:  dailyPrice.TickerSymbol,
		Date:          dailyPrice.Date,
		OpenPrice:     open,
		ClosePrice:    closePrice,
		HighPrice:     high,
		LowPrice:      low,
		AdjClosePrice: adjclose,
		Volume:        uint64(dailyPrice.Volume),
		CreatedAt:     dailyPrice.CreatedAt,
		UpdatedAt:     dailyPrice.UpdatedAt,
	}
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StockBrandsDailyPriceForAnalyzeRepositoryImpl struct {
	query *genQuery.Query
}

func NewStockBrandsDailyPriceForAnalyzeRepositoryImpl(db *gorm.DB) repositories.StockBrandsDailyPriceForAnalyzeRepository {
	return &StockBrandsDailyPriceForAnalyzeRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (si *StockBrandsDailyPriceForAnalyzeRepositoryImpl) CreateStockBrandDailyPriceForAnalyze(ctx context.Context, dailyPrices []*models.StockBrandDailyPriceForAnalyze) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if err := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).
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
		return errors.Wrap(err, "CreateStockBrandsDailyPriceForAnalyze error")
	}
	return nil
}

func (si *StockBrandsDailyPriceForAnalyzeRepositoryImpl) DeleteBySymbols(ctx context.Context, deleteSymbols []string) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if _, err := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).
		Where(tx.StockBrandsDailyPriceForAnalyze.TickerSymbol.In(deleteSymbols...)).
		Delete(); err != nil {
		return errors.Wrap(err, "StockBrandsDailyPriceForAnalyze.deleteSymbols error")
	}
	return nil
}

// func (si *StockBrandsDailyPriceForAnalyzeRepositoryImpl) convertToDomainModel(dailyPriceDB *genModel.StockBrandsDailyPriceForAnalyze) *models.StockBrandDailyPriceForAnalyze {
// 	if dailyPriceDB == nil {
// 		return nil
// 	}

// 	return &models.StockBrandDailyPriceForAnalyze{
// 		ID:           dailyPriceDB.ID,
// 		TickerSymbol: dailyPriceDB.TickerSymbol,
// 		Date:         dailyPriceDB.Date,
// 		Open:         decimal.NewFromFloat(dailyPriceDB.OpenPrice),
// 		Close:        decimal.NewFromFloat(dailyPriceDB.ClosePrice),
// 		High:         decimal.NewFromFloat(dailyPriceDB.HighPrice),
// 		Low:          decimal.NewFromFloat(dailyPriceDB.LowPrice),
// 		Adjclose:     decimal.NewFromFloat(dailyPriceDB.AdjClosePrice),
// 		Volume:       int64(dailyPriceDB.Volume),
// 		CreatedAt:    dailyPriceDB.CreatedAt,
// 		UpdatedAt:    dailyPriceDB.UpdatedAt,
// 	}
// }

func (si *StockBrandsDailyPriceForAnalyzeRepositoryImpl) convertToDBModels(dailyPrices []*models.StockBrandDailyPriceForAnalyze) []*genModel.StockBrandsDailyPriceForAnalyze {
	var dailyPricesDB []*genModel.StockBrandsDailyPriceForAnalyze
	for _, v := range dailyPrices {
		dailyPricesDB = append(dailyPricesDB, si.convertToDBModel(v))
	}
	return dailyPricesDB
}

func (si *StockBrandsDailyPriceForAnalyzeRepositoryImpl) convertToDBModel(dailyPrice *models.StockBrandDailyPriceForAnalyze) *genModel.StockBrandsDailyPriceForAnalyze {
	open, _ := dailyPrice.Open.Round(4).Float64()
	close, _ := dailyPrice.Close.Round(4).Float64()
	high, _ := dailyPrice.High.Round(4).Float64()
	low, _ := dailyPrice.Low.Round(4).Float64()
	adjclose, _ := dailyPrice.Adjclose.Round(4).Float64()
	return &genModel.StockBrandsDailyPriceForAnalyze{
		ID:            dailyPrice.ID,
		TickerSymbol:  dailyPrice.TickerSymbol,
		Date:          dailyPrice.Date,
		OpenPrice:     open,
		ClosePrice:    close,
		HighPrice:     high,
		LowPrice:      low,
		AdjClosePrice: adjclose,
		Volume:        uint64(dailyPrice.Volume),
		CreatedAt:     dailyPrice.CreatedAt,
		UpdatedAt:     dailyPrice.UpdatedAt,
	}
}

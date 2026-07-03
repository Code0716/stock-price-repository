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
		Where(
			tx.StockBrandsDailyPrice.
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

	// ソート順の適用（デフォルトは昇順）
	if filter.DateOrder != nil && *filter.DateOrder == models.SortOrderDesc {
		q = q.Order(tx.StockBrandsDailyPrice.Date.Desc())
	} else {
		q = q.Order(tx.StockBrandsDailyPrice.Date)
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

// ListRangePricesBySymbols 複数銘柄の期間中日足を一括取得する（シグナル精度評価用）
func (si *StockBrandsDailyPriceRepositoryImpl) ListRangePricesBySymbols(ctx context.Context, filter models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if len(filter.Symbols) == 0 {
		return []*models.StockBrandDailyPrice{}, nil
	}

	q := tx.WithContext(ctx).
		StockBrandsDailyPrice.
		Where(tx.StockBrandsDailyPrice.TickerSymbol.In(filter.Symbols...))

	if filter.DateFrom != nil {
		dateOnly := time.Date(filter.DateFrom.Year(), filter.DateFrom.Month(), filter.DateFrom.Day(), 0, 0, 0, 0, filter.DateFrom.Location())
		q = q.Where(tx.StockBrandsDailyPrice.Date.Gte(dateOnly))
	}
	if filter.DateTo != nil {
		dateOnly := time.Date(filter.DateTo.Year(), filter.DateTo.Month(), filter.DateTo.Day(), 0, 0, 0, 0, filter.DateTo.Location())
		q = q.Where(tx.StockBrandsDailyPrice.Date.Lte(dateOnly))
	}

	q = q.Order(tx.StockBrandsDailyPrice.TickerSymbol).Order(tx.StockBrandsDailyPrice.Date)

	rows, err := q.Find()
	if err != nil {
		return nil, errors.Wrap(err, "StockBrandsDailyPriceRepositoryImpl.ListRangePricesBySymbols error")
	}

	prices := make([]*models.StockBrandDailyPrice, 0, len(rows))
	for _, r := range rows {
		prices = append(prices, si.convertToDomainModel(r))
	}
	return prices, nil
}

// ListRecentTradingDates onOrBefore以前の直近の営業日（データが存在する日）を新しい順にlimit件取得する（クイズのユニバース選定用）。
func (si *StockBrandsDailyPriceRepositoryImpl) ListRecentTradingDates(ctx context.Context, onOrBefore time.Time, limit int) ([]time.Time, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	dateOnly := time.Date(onOrBefore.Year(), onOrBefore.Month(), onOrBefore.Day(), 0, 0, 0, 0, onOrBefore.Location())

	var dates []time.Time
	if err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Where(tx.StockBrandsDailyPrice.Date.Lte(dateOnly)).
		Distinct(tx.StockBrandsDailyPrice.Date).
		Order(tx.StockBrandsDailyPrice.Date.Desc()).
		Limit(limit).
		Pluck(tx.StockBrandsDailyPrice.Date, &dates); err != nil {
		return nil, errors.Wrap(err, "StockBrandsDailyPriceRepositoryImpl.ListRecentTradingDates error")
	}

	return dates, nil
}

// ListPricesByDateRange 期間中の全銘柄の日足を取得する（クイズのユニバース選定用）。
func (si *StockBrandsDailyPriceRepositoryImpl) ListPricesByDateRange(ctx context.Context, from, to time.Time) ([]*models.StockBrandDailyPrice, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	dateFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	dateTo := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())

	rows, err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Where(tx.StockBrandsDailyPrice.Date.Gte(dateFrom)).
		Where(tx.StockBrandsDailyPrice.Date.Lte(dateTo)).
		Order(tx.StockBrandsDailyPrice.TickerSymbol).
		Order(tx.StockBrandsDailyPrice.Date).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "StockBrandsDailyPriceRepositoryImpl.ListPricesByDateRange error")
	}

	prices := make([]*models.StockBrandDailyPrice, 0, len(rows))
	for _, r := range rows {
		prices = append(prices, si.convertToDomainModel(r))
	}
	return prices, nil
}

// FindNextTradingDate afterより後の直近の営業日を1件取得する（存在しなければnil。クイズ採点用）。
func (si *StockBrandsDailyPriceRepositoryImpl) FindNextTradingDate(ctx context.Context, after time.Time) (*time.Time, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	dateOnly := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location())

	price, err := tx.StockBrandsDailyPrice.WithContext(ctx).
		Where(tx.StockBrandsDailyPrice.Date.Gt(dateOnly)).
		Order(tx.StockBrandsDailyPrice.Date).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "StockBrandsDailyPriceRepositoryImpl.FindNextTradingDate error")
	}

	return &price.Date, nil
}

func (si *StockBrandsDailyPriceRepositoryImpl) convertToDomainModel(dailyPriceDB *genModel.StockBrandsDailyPrice) *models.StockBrandDailyPrice {
	if dailyPriceDB == nil {
		return nil
	}

	return &models.StockBrandDailyPrice{
		ID:           dailyPriceDB.ID,
		StockBrandID: dailyPriceDB.StockBrandID,
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

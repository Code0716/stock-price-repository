//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type FinStatementRepositoryImpl struct {
	db *gorm.DB
}

func NewFinStatementRepositoryImpl(db *gorm.DB) repositories.FinStatementRepository {
	return &FinStatementRepositoryImpl{db: db}
}

type finStatementRow struct {
	ID                      string           `gorm:"column:id"`
	TickerSymbol            string           `gorm:"column:ticker_symbol"`
	StockBrandID            *string          `gorm:"column:stock_brand_id"`
	DisclosedDate           time.Time        `gorm:"column:disclosed_date"`
	FiscalYearEnd           *time.Time       `gorm:"column:fiscal_year_end"`
	TypeOfDocument          string           `gorm:"column:type_of_document"`
	TypeOfCurrentPeriod     string           `gorm:"column:type_of_current_period"`
	NetSales                *decimal.Decimal `gorm:"column:net_sales"`
	OperatingProfit         *decimal.Decimal `gorm:"column:operating_profit"`
	OrdinaryProfit          *decimal.Decimal `gorm:"column:ordinary_profit"`
	Profit                  *decimal.Decimal `gorm:"column:profit"`
	EarningsPerShare        *decimal.Decimal `gorm:"column:earnings_per_share"`
	BookValuePerShare       *decimal.Decimal `gorm:"column:book_value_per_share"`
	ForecastNetSales        *decimal.Decimal `gorm:"column:forecast_net_sales"`
	ForecastOperatingProfit *decimal.Decimal `gorm:"column:forecast_operating_profit"`
	ForecastProfit          *decimal.Decimal `gorm:"column:forecast_profit"`
	ForecastEPS             *decimal.Decimal `gorm:"column:forecast_eps"`
	CreatedAt               time.Time        `gorm:"column:created_at"`
	UpdatedAt               time.Time        `gorm:"column:updated_at"`
}

func (finStatementRow) TableName() string { return "fin_statement" }

func (r *FinStatementRepositoryImpl) Upsert(ctx context.Context, statements []*models.FinStatement) error {
	if len(statements) == 0 {
		return nil
	}

	rows := make([]*finStatementRow, 0, len(statements))
	for _, s := range statements {
		rows = append(rows, &finStatementRow{
			ID:                      s.ID,
			TickerSymbol:            s.TickerSymbol,
			StockBrandID:            s.StockBrandID,
			DisclosedDate:           s.DisclosedDate,
			FiscalYearEnd:           s.FiscalYearEnd,
			TypeOfDocument:          s.TypeOfDocument,
			TypeOfCurrentPeriod:     s.TypeOfCurrentPeriod,
			NetSales:                s.NetSales,
			OperatingProfit:         s.OperatingProfit,
			OrdinaryProfit:          s.OrdinaryProfit,
			Profit:                  s.Profit,
			EarningsPerShare:        s.EarningsPerShare,
			BookValuePerShare:       s.BookValuePerShare,
			ForecastNetSales:        s.ForecastNetSales,
			ForecastOperatingProfit: s.ForecastOperatingProfit,
			ForecastProfit:          s.ForecastProfit,
			ForecastEPS:             s.ForecastEPS,
			CreatedAt:               s.CreatedAt,
			UpdatedAt:               s.UpdatedAt,
		})
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "ticker_symbol"}, {Name: "disclosed_date"}, {Name: "type_of_document"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"net_sales", "operating_profit", "ordinary_profit", "profit",
				"earnings_per_share", "book_value_per_share",
				"forecast_net_sales", "forecast_operating_profit", "forecast_profit", "forecast_eps",
				"fiscal_year_end", "type_of_current_period", "updated_at",
			}),
		}).
		Create(&rows).Error; err != nil {
		return errors.Wrap(err, "FinStatementRepositoryImpl.Upsert error")
	}
	return nil
}

func (r *FinStatementRepositoryImpl) FindBySymbol(ctx context.Context, filter *models.FinStatementFilter) ([]*models.FinStatement, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 8
	}

	var rows []*finStatementRow
	if err := r.db.WithContext(ctx).
		Table("fin_statement").
		Where("ticker_symbol = ?", filter.TickerSymbol).
		Order("disclosed_date DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "FinStatementRepositoryImpl.FindBySymbol error")
	}

	result := make([]*models.FinStatement, 0, len(rows))
	for _, row := range rows {
		result = append(result, r.convertToDomainModel(row))
	}
	return result, nil
}

func (r *FinStatementRepositoryImpl) convertToDomainModel(row *finStatementRow) *models.FinStatement {
	return &models.FinStatement{
		ID:                      row.ID,
		TickerSymbol:            row.TickerSymbol,
		StockBrandID:            row.StockBrandID,
		DisclosedDate:           row.DisclosedDate,
		FiscalYearEnd:           row.FiscalYearEnd,
		TypeOfDocument:          row.TypeOfDocument,
		TypeOfCurrentPeriod:     row.TypeOfCurrentPeriod,
		NetSales:                row.NetSales,
		OperatingProfit:         row.OperatingProfit,
		OrdinaryProfit:          row.OrdinaryProfit,
		Profit:                  row.Profit,
		EarningsPerShare:        row.EarningsPerShare,
		BookValuePerShare:       row.BookValuePerShare,
		ForecastNetSales:        row.ForecastNetSales,
		ForecastOperatingProfit: row.ForecastOperatingProfit,
		ForecastProfit:          row.ForecastProfit,
		ForecastEPS:             row.ForecastEPS,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}

package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type FinStatement struct {
	ID                   string
	TickerSymbol         string
	StockBrandID         *string
	DisclosedDate        time.Time
	FiscalYearEnd        *time.Time
	TypeOfDocument       string
	TypeOfCurrentPeriod  string
	NetSales             *decimal.Decimal
	OperatingProfit      *decimal.Decimal
	OrdinaryProfit       *decimal.Decimal
	Profit               *decimal.Decimal
	EarningsPerShare     *decimal.Decimal
	BookValuePerShare    *decimal.Decimal
	ForecastNetSales     *decimal.Decimal
	ForecastOperatingProfit *decimal.Decimal
	ForecastProfit       *decimal.Decimal
	ForecastEPS          *decimal.Decimal
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type FinStatementFilter struct {
	TickerSymbol string
	Limit        int
}

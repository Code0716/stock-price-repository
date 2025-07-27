package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// StockBrandDailyPriceForAnalyze
type StockBrandDailyPriceForAnalyze struct {
	ID           string
	TickerSymbol string
	Date         time.Time
	High         decimal.Decimal
	Low          decimal.Decimal
	Open         decimal.Decimal
	Close        decimal.Decimal
	Volume       int64
	Adjclose     decimal.Decimal
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewStockBrandDailyPriceForAnalyze(
	id string,
	date time.Time,
	tickerSymbol string,
	high decimal.Decimal,
	low decimal.Decimal,
	open decimal.Decimal,
	close decimal.Decimal,
	volume int64,
	adjclose decimal.Decimal,
	createdAt time.Time,
	updatedAt time.Time,
) *StockBrandDailyPriceForAnalyze {
	return &StockBrandDailyPriceForAnalyze{
		ID:           id,
		Date:         date,
		TickerSymbol: tickerSymbol,
		High:         high,
		Low:          low,
		Open:         open,
		Close:        close,
		Volume:       volume,
		Adjclose:     adjclose,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

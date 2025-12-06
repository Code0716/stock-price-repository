package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// StockBrandDailyPriceForAnalyze
type StockBrandDailyPriceForAnalyze struct {
	ID           string          `json:"id"`
	TickerSymbol string          `json:"tickerSymbol"`
	Date         time.Time       `json:"date"`
	High         decimal.Decimal `json:"high"`
	Low          decimal.Decimal `json:"low"`
	Open         decimal.Decimal `json:"open"`
	Close        decimal.Decimal `json:"close"`
	Volume       int64           `json:"volume"`
	Adjclose     decimal.Decimal `json:"adjClose"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

func NewStockBrandDailyPriceForAnalyze(
	id string,
	date time.Time,
	tickerSymbol string,
	high decimal.Decimal,
	low decimal.Decimal,
	open decimal.Decimal,
	closePrice decimal.Decimal,
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
		Close:        closePrice,
		Volume:       volume,
		Adjclose:     adjclose,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

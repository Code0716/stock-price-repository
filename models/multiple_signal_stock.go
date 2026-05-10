package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type MultipleSignalStock struct {
	StockBrandID string          `json:"stockBrandId"`
	Name         string          `json:"name"`
	TickerSymbol string          `json:"tickerSymbol"`
	Date         time.Time       `json:"date"`
	Methods      []string        `json:"methods"`
	SignalCount   int             `json:"signalCount"`
	CurrentPrice decimal.Decimal `json:"currentPrice"`
}

type MultipleSignalStockFilter struct {
	Date   *time.Time
	Cursor string
	Limit  int
}

type PaginatedMultipleSignalStocks struct {
	Stocks     []*MultipleSignalStock
	NextCursor *string
	Limit      int
}

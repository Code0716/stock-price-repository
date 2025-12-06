package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type StockBrandDailyPrice struct {
	ID           string          `json:"id"`
	StockBrandID string          `json:"stockBrandId"`
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

func NewStockBrandDailyPrice(
	id string,
	stockBrandID string,
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
) *StockBrandDailyPrice {
	return &StockBrandDailyPrice{
		ID:           id,
		StockBrandID: stockBrandID,
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

// SortOrder ソート順を表す型
type SortOrder string

const (
	// SortOrderAsc 昇順
	SortOrderAsc SortOrder = "asc"
	// SortOrderDesc 降順
	SortOrderDesc SortOrder = "desc"
)

// ListDailyPricesBySymbolFilter 期間中の日足をsymbolから取得する。
type ListDailyPricesBySymbolFilter struct {
	TickerSymbol string
	DateFrom     *time.Time
	DateTo       *time.Time
	DateOrder    *SortOrder // 日付のソート順（nilの場合は昇順）
}

// ListRangePricesBySymbolsFilter 期間中の日足を複数のsymbolから取得する。
type ListRangePricesBySymbolsFilter struct {
	Symbols  []string
	DateFrom *time.Time
	DateTo   *time.Time
}

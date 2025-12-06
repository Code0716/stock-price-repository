package models

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

type IndexStockAverageDailyPrices []*IndexStockAverageDailyPrice
type IndexStockAverageDailyPrice struct {
	Date      time.Time       `json:"date"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	Volume    int64           `json:"volume"`
	Adjclose  decimal.Decimal `json:"adjClose"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

func NewIndexStockAverageDailyPrice(
	Date time.Time,
	High decimal.Decimal,
	Low decimal.Decimal,
	Open decimal.Decimal,
	Close decimal.Decimal,
	Volume int64,
	Adjclose decimal.Decimal,
	CreatedAt time.Time,
	UpdatedAt time.Time,
) *IndexStockAverageDailyPrice {
	return &IndexStockAverageDailyPrice{
		Date:      Date,
		High:      High,
		Low:       Low,
		Open:      Open,
		Close:     Close,
		Volume:    Volume,
		Adjclose:  Adjclose,
		CreatedAt: CreatedAt,
		UpdatedAt: UpdatedAt,
	}
}

type MovingAverageAndDate struct {
	Date    time.Time
	Average decimal.Decimal
}

func NewMovingAverageAndDate(date time.Time, average decimal.Decimal) *MovingAverageAndDate {
	return &MovingAverageAndDate{
		Date:    date,
		Average: average,
	}
}

func SortMovingAverageAndDatesDESC(items []*MovingAverageAndDate) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Date.After(items[j].Date)
	})
}

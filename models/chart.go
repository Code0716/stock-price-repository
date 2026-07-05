package models

import (
	"github.com/shopspring/decimal"
)

// ChartCandle チャート用の1本の日足。
type ChartCandle struct {
	Date   string          `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume int64           `json:"volume"`
}

// ChartMAPoint 移動平均線の1点。
type ChartMAPoint struct {
	Date  string          `json:"date"`
	Value decimal.Decimal `json:"value"`
}

// DailyPriceChart GET /daily-prices/chart のレスポンス。
type DailyPriceChart struct {
	Candles []*ChartCandle  `json:"candles"`
	MA5     []*ChartMAPoint `json:"ma5"`
	MA25    []*ChartMAPoint `json:"ma25"`
	MA75    []*ChartMAPoint `json:"ma75"`
}

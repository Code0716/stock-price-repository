package models

import "github.com/shopspring/decimal"

// TechnicalIndicatorPoint 1日分のテクニカル指標値。
// 確定していない（ウォームアップ期間中）の指標は nil。
type TechnicalIndicatorPoint struct {
	Date    string           `json:"date"`
	Close   *decimal.Decimal `json:"close"`
	ATR     *decimal.Decimal `json:"atr"`
	StochK  *decimal.Decimal `json:"stochK"`
	StochD  *decimal.Decimal `json:"stochD"`
	ADX     *decimal.Decimal `json:"adx"`
	PlusDI  *decimal.Decimal `json:"plusDI"`
	MinusDI *decimal.Decimal `json:"minusDI"`
	OBV     *decimal.Decimal `json:"obv"`
	VWAP    *decimal.Decimal `json:"vwap"`
	Tenkan  *decimal.Decimal `json:"tenkan"`
	Kijun   *decimal.Decimal `json:"kijun"`
	SenkouA *decimal.Decimal `json:"senkouA"`
	SenkouB *decimal.Decimal `json:"senkouB"`
	Chikou  *decimal.Decimal `json:"chikou"`
}

// SupportResistanceLevel サポート/レジスタンスレベル。
type SupportResistanceLevel struct {
	Price    *decimal.Decimal `json:"price"`
	Kind     string           `json:"kind"`     // "support" | "resistance"
	Touches  int              `json:"touches"`
	LastDate string           `json:"lastDate"` // YYYY-MM-DD
}

// TechnicalIndicators 銘柄の指定期間テクニカル指標時系列。
type TechnicalIndicators struct {
	Symbol                  string                    `json:"symbol"`
	From                    string                    `json:"from"`
	To                      string                    `json:"to"`
	TradingDays             int                       `json:"tradingDays"`
	Points                  []TechnicalIndicatorPoint `json:"points"`
	FuturePoints            []TechnicalIndicatorPoint `json:"futurePoints"`
	SupportResistanceLevels []SupportResistanceLevel  `json:"supportResistanceLevels"`
}

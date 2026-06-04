package models

import "github.com/shopspring/decimal"

// Valuation 銘柄の評価指標。最新終値と直近財務データから算出する。
// 算出に必要なデータが無い（赤字EPS・データ欠落等）場合は該当フィールドが nil になる。
type Valuation struct {
	Symbol      string           `json:"symbol"`
	Close       *decimal.Decimal `json:"close"`       // 最新終値（参考）
	PriceDate   string           `json:"priceDate"`   // 終値の日付 YYYY-MM-DD
	PER         *decimal.Decimal `json:"per"`         // 株価 ÷ 実績(通期FY)EPS
	ForwardPER  *decimal.Decimal `json:"forwardPer"`  // 株価 ÷ 通期予想EPS
	PBR         *decimal.Decimal `json:"pbr"`         // 株価 ÷ BPS
	ROE         *decimal.Decimal `json:"roe"`         // 実績EPS ÷ BPS（簡易値。比率: 0.12=12%）
	TrailingEPS *decimal.Decimal `json:"trailingEps"` // 実績(FY)EPS（算出根拠）
	ForecastEPS *decimal.Decimal `json:"forecastEps"` // 通期予想EPS
	BPS         *decimal.Decimal `json:"bps"`         // 1株あたり純資産
	// FiscalPeriod 実績EPSの決算期末（例 "2025-03"）。trailingEPSが無い場合は ""。
	FiscalPeriod string `json:"fiscalPeriod"`
}

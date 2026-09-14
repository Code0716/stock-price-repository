package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// DailyAvoidSeverity 避けるべき銘柄の重大度。
type DailyAvoidSeverity string

const (
	// DailyAvoidSeverityHigh ボラティリティが流動性ユニバース内で上位10%以内。
	DailyAvoidSeverityHigh DailyAvoidSeverity = "high"
	// DailyAvoidSeverityElevated ボラティリティが流動性ユニバース内で上位10〜20%。
	DailyAvoidSeverityElevated DailyAvoidSeverity = "elevated"
)

// DailyAvoidReasonHighVolatility 判定理由: 直近12ヶ月の実現ボラティリティが高い。
// 現状はこれのみ。将来別の判定軸を追加する場合に備えた識別子。
const DailyAvoidReasonHighVolatility = "high_volatility"

// DailyAvoidStock daily_avoid_stock のドメインモデル。
// 流動性ユニバース（平均売買代金1億円/日以上）内で、直近12ヶ月の実現ボラティリティが
// 上位20%に入る銘柄を「避けるべき銘柄」として日次で保存する。daily_stock_pick とは独立。
type DailyAvoidStock struct {
	AsOfDate             time.Time
	StockBrandID         string
	TickerSymbol         string
	Name                 string // DB非保存。usecase側でStockBrandRepositoryから解決する
	AvoidRank            int
	Severity             DailyAvoidSeverity
	Reason               string
	RuleVersion          string
	Volatility12M        decimal.Decimal
	VolatilityPercentile decimal.Decimal
	UniverseSize         int
	ThresholdVolatility  decimal.Decimal
	AvgTradingValue      decimal.Decimal
	BaseClosePrice       decimal.Decimal
	Sector33CodeName     string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// DailyAvoidStockItem API レスポンス用の1銘柄分。
type DailyAvoidStockItem struct {
	AvoidRank            int                `json:"avoidRank"`
	StockBrandID         string             `json:"stockBrandId"`
	TickerSymbol         string             `json:"tickerSymbol"`
	Name                 string             `json:"name"`
	Severity             DailyAvoidSeverity `json:"severity"`
	Reason               string             `json:"reason"`
	Volatility12M        decimal.Decimal    `json:"volatility12m"`
	VolatilityPercentile decimal.Decimal    `json:"volatilityPercentile"`
	AvgTradingValue      decimal.Decimal    `json:"avgTradingValue"`
	BaseClosePrice       decimal.Decimal    `json:"baseClosePrice"`
	Sector33CodeName     string             `json:"sector33CodeName"`
}

// DailyAvoidStockSummary GET /daily-avoid-stocks の集計サマリ。
type DailyAvoidStockSummary struct {
	FlaggedCount         int             `json:"flaggedCount"`
	HighCount            int             `json:"highCount"`
	ElevatedCount        int             `json:"elevatedCount"`
	MaxVolatility        decimal.Decimal `json:"maxVolatility"`
	MinFlaggedVolatility decimal.Decimal `json:"minFlaggedVolatility"`
}

// DailyAvoidStockDay GET /daily-avoid-stocks のレスポンス。
// 該当日のデータが無い場合は AsOfDate が null、Items が空配列になる（エラーにはしない）。
type DailyAvoidStockDay struct {
	AsOfDate            *string                `json:"asOfDate"`
	RuleVersion         string                 `json:"ruleVersion"`
	UniverseSize        int                    `json:"universeSize"`
	ThresholdVolatility decimal.Decimal        `json:"thresholdVolatility"`
	Summary             DailyAvoidStockSummary `json:"summary"`
	Items               []*DailyAvoidStockItem `json:"items"`
}

// DailyAvoidStockDates GET /daily-avoid-stocks/dates のレスポンス。
type DailyAvoidStockDates struct {
	Dates []string `json:"dates"` // YYYY-MM-DD、新しい順
}

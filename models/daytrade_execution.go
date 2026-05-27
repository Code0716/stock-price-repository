package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type DaytradeExecution struct {
	ID           uint64          `json:"id"`
	ExecutedOn   time.Time       `json:"executedOn"`
	TradeKind    string          `json:"tradeKind"`
	MarginKind   string          `json:"marginKind"`
	TickerSymbol string          `json:"tickerSymbol"`
	BrandName    string          `json:"brandName"`
	Quantity     uint32          `json:"quantity"`
	TradeAmount  int64           `json:"tradeAmount"`
	UnitPrice    decimal.Decimal `json:"unitPrice"`
	AverageCost  decimal.Decimal `json:"averageCost"`
	ProfitLoss   int64           `json:"profitLoss"`
	OccurrenceNo uint32          `json:"occurrenceNo"`
	Source       string          `json:"source"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type DaytradeSummaryGranularity string

const (
	DaytradeSummaryGranularityDaily   DaytradeSummaryGranularity = "daily"
	DaytradeSummaryGranularityMonthly DaytradeSummaryGranularity = "monthly"
	DaytradeSummaryGranularityYearly  DaytradeSummaryGranularity = "yearly"
	DaytradeSummaryGranularityAll     DaytradeSummaryGranularity = "all"
)

func (g DaytradeSummaryGranularity) Valid() bool {
	switch g {
	case DaytradeSummaryGranularityDaily,
		DaytradeSummaryGranularityMonthly,
		DaytradeSummaryGranularityYearly,
		DaytradeSummaryGranularityAll:
		return true
	}
	return false
}

type DaytradeSummaryBucket struct {
	BucketDate  *string `json:"date"`
	ProfitLoss  int64   `json:"profitLoss"`
	TradeCount  int     `json:"tradeCount"`
	GrossProfit int64   `json:"grossProfit"`
	GrossLoss   int64   `json:"grossLoss"`
	WinCount    int     `json:"winCount"`
	LossCount   int     `json:"lossCount"`
}

type DaytradeImportResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
	Deleted  int `json:"deleted"`
	TotalRow int `json:"totalRow"`
}

// DaytradeSymbolSummary 銘柄毎のデイトレ損益集計
type DaytradeSymbolSummary struct {
	TickerSymbol string `json:"tickerSymbol"`
	BrandName    string `json:"brandName"`
	ProfitLoss   int64  `json:"profitLoss"`
	TradeCount   int    `json:"tradeCount"`
	GrossProfit  int64  `json:"grossProfit"`
	GrossLoss    int64  `json:"grossLoss"`
	WinCount     int    `json:"winCount"`
	LossCount    int    `json:"lossCount"`
}

// DaytradeStatsAggregate スカラー集計の内部受け皿（MAX/MIN 含む）
type DaytradeStatsAggregate struct {
	ProfitLoss  int64
	TradeCount  int
	GrossProfit int64
	GrossLoss   int64
	WinCount    int
	LossCount   int
	MaxProfit   int64 // MAX(profit_loss)。データなしなら 0
	MaxLoss     int64 // MIN(profit_loss)。データなしなら 0
}

// DaytradePeriodStats 期間統計（/daytrade/stats API レスポンス）
type DaytradePeriodStats struct {
	ProfitLoss    int64 `json:"profitLoss"`
	TradeCount    int   `json:"tradeCount"`
	GrossProfit   int64 `json:"grossProfit"`
	GrossLoss     int64 `json:"grossLoss"`
	WinCount      int   `json:"winCount"`
	LossCount     int   `json:"lossCount"`
	MaxProfit     int64 `json:"maxProfit"`
	MaxLoss       int64 `json:"maxLoss"`
	MaxDrawdown   int64 `json:"maxDrawdown"`
	MaxRunup      int64 `json:"maxRunup"`
	MaxLossStreak int   `json:"maxLossStreak"`
}

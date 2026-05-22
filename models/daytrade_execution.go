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
	BucketDate *string `json:"date"`
	ProfitLoss int64   `json:"profitLoss"`
	TradeCount int     `json:"tradeCount"`
}

type DaytradeImportResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
	TotalRow int `json:"totalRow"`
}

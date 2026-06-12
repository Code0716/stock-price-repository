package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Sector33AverageDailyPrice sector_33_average_daily_price テーブルのドメインモデル
type Sector33AverageDailyPrice struct {
	Date       time.Time
	SectorCode string
	Open       decimal.Decimal
	Close      decimal.Decimal
	High       decimal.Decimal
	Low        decimal.Decimal
	Adjclose   decimal.Decimal
}

// SectorPerformanceItem 1業種のパフォーマンス指標
type SectorPerformanceItem struct {
	SectorCode   string           `json:"sectorCode"`
	SectorName   string           `json:"sectorName"`
	PeriodReturn *decimal.Decimal `json:"periodReturn"`
	Return5d     *decimal.Decimal `json:"return5d"`
	Return25d    *decimal.Decimal `json:"return25d"`
	LatestClose  *decimal.Decimal `json:"latestClose"`
	LatestDate   string           `json:"latestDate"`
}

// SectorPerformance GET /sector-performance のレスポンス全体
type SectorPerformance struct {
	From    string                   `json:"from"`
	To      string                   `json:"to"`
	Sectors []*SectorPerformanceItem `json:"sectors"`
}

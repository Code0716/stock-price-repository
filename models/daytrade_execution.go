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

// DaytradeTradeApprox は「銘柄×日×売買方向」で集約した1トレード近似
type DaytradeTradeApprox struct {
	TickerSymbol string
	BrandName    string
	ExecutedOn   time.Time
	Direction    string // tradeKind が空なら marginKind、それ以外は tradeKind
	ProfitLoss   int64
	TradeAmount  int64
	WinCount     int // 集約行のうち profit_loss > 0 の件数
	LossCount    int // 集約行のうち profit_loss < 0 の件数
}

// DaytradeInsights デイトレ反省ダッシュボード（/daytrade/insights API レスポンス）
type DaytradeInsights struct {
	LossConcentration DaytradeLossConcentration `json:"lossConcentration"`
	FavoriteTraps     []DaytradeFavoriteTrap    `json:"favoriteTraps"`
}

// DaytradeLossConcentration 大損寄与率（パレート分析）
type DaytradeLossConcentration struct {
	TotalLoss   int64                `json:"totalLoss"`   // 負けトレード損失合計（絶対値）
	Top1Ratio   float64              `json:"top1Ratio"`   // 上位1件が総損失に占める割合
	Top3Ratio   float64              `json:"top3Ratio"`
	Top5Ratio   float64              `json:"top5Ratio"`
	WorstTrades []DaytradeWorstTrade `json:"worstTrades"` // 損失上位5件（損失大きい順）
}

// DaytradeWorstTrade 損失上位トレード
type DaytradeWorstTrade struct {
	TickerSymbol string `json:"tickerSymbol"`
	BrandName    string `json:"brandName"`
	ExecutedOn   string `json:"executedOn"` // YYYY-MM-DD
	Direction    string `json:"direction"`
	ProfitLoss   int64  `json:"profitLoss"`
}

// DaytradeFavoriteTrap 惚れ込み検出（頻繁に取引するが期待値マイナスの銘柄）
type DaytradeFavoriteTrap struct {
	TickerSymbol string  `json:"tickerSymbol"`
	BrandName    string  `json:"brandName"`
	TradeCount   int     `json:"tradeCount"` // 集約後トレード数
	TotalPnl     int64   `json:"totalPnl"`
	Expectancy   float64 `json:"expectancy"` // TotalPnl / TradeCount
	WinRate      float64 `json:"winRate"`
}

// DaytradeTradeNoteRecord はリポジトリ層が扱うトレード注釈の内部モデル
type DaytradeTradeNoteRecord struct {
	TickerSymbol      string           // 近似キー
	ExecutedOn        time.Time        // 近似キー
	Direction         string           // 近似キー（正規化済み）
	Memo              string
	Tags              []string
	DeclaredStopPrice *decimal.Decimal
}

// DaytradeTradeNote はAPIレスポンス中の注釈部分
type DaytradeTradeNote struct {
	Memo              string           `json:"memo"`
	Tags              []string         `json:"tags"`
	DeclaredStopPrice *decimal.Decimal `json:"declaredStopPrice"`
}

// DaytradeTradeWithNote は近似トレード＋注釈（GET /daytrade/trades の1行）
type DaytradeTradeWithNote struct {
	TickerSymbol string             `json:"tickerSymbol"`
	BrandName    string             `json:"brandName"`
	ExecutedOn   string             `json:"executedOn"` // YYYY-MM-DD
	Direction    string             `json:"direction"`
	ProfitLoss   int64              `json:"profitLoss"`
	TradeAmount  int64              `json:"tradeAmount"`
	Note         *DaytradeTradeNote `json:"note"` // 未注釈なら null
}

// DaytradeTagStat はタグ別損益集計（GET /daytrade/tag-stats の1行）
type DaytradeTagStat struct {
	Tag        string  `json:"tag"`
	TradeCount int     `json:"tradeCount"`
	TotalPnl   int64   `json:"totalPnl"`
	Expectancy float64 `json:"expectancy"` // TotalPnl / TradeCount
	WinRate    float64 `json:"winRate"`
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

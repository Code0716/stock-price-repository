package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SignalPerformanceFilter GET /signal-performance のクエリパラメータ
type SignalPerformanceFilter struct {
	From   time.Time
	To     time.Time
	Method string // 任意：完全一致。空文字なら全手法
	Action string // 任意："Buy" / "Sell"。空文字なら両方
}

// HorizonStats 1手法 × 1 horizon の統計
type HorizonStats struct {
	EvaluatedCount int             `json:"evaluatedCount"`
	WinCount       int             `json:"winCount"`
	WinRate        decimal.Decimal `json:"winRate"`
	AvgReturn      decimal.Decimal `json:"avgReturn"`
	MedianReturn   decimal.Decimal `json:"medianReturn"`
	BestReturn     decimal.Decimal `json:"bestReturn"`
	WorstReturn    decimal.Decimal `json:"worstReturn"`
}

// SignalPerformanceSummary 1手法の集計サマリ
type SignalPerformanceSummary struct {
	Method       string                `json:"method"`
	SignalCount  int                   `json:"signalCount"`
	SkippedCount int                   `json:"skippedCount"`
	Stats        map[int]*HorizonStats `json:"stats"` // key: 5 / 10 / 20
}

// EvaluatedSignal 1シグナルの明細
type EvaluatedSignal struct {
	TickerSymbol string                   `json:"tickerSymbol"`
	Name         string                   `json:"name"`
	Method       string                   `json:"method"`
	Date         time.Time                `json:"date"`
	Action       string                   `json:"action"`
	BasePrice    decimal.Decimal          `json:"basePrice"`
	Score        *decimal.Decimal         `json:"score"`
	SignalRank   *int                     `json:"signalRank"`
	Memo         *string                  `json:"memo"`
	Returns      map[int]*decimal.Decimal `json:"returns"` // key: 5/10/20, nil=未到来
}

// BandSummary rank帯・score四分位など、シグナルの帯別集計
type BandSummary struct {
	Band        string                `json:"band"`        // 例: "1-3", "4-10", "11+", "Q1".."Q4"
	SignalCount int                   `json:"signalCount"` // 帯に属するシグナル数（skip含む）
	Stats       map[int]*HorizonStats `json:"stats"`
}

// SignalPerformance API レスポンス全体
type SignalPerformance struct {
	From           time.Time                  `json:"from"`
	To             time.Time                  `json:"to"`
	Horizons       []int                      `json:"horizons"`
	Summaries      []*SignalPerformanceSummary `json:"summaries"`
	Signals        []*EvaluatedSignal         `json:"signals"`        // method 指定時のみ、非 nil
	RankBands      []*BandSummary             `json:"rankBands"`      // method 指定時のみ非nil
	ScoreQuartiles []*BandSummary             `json:"scoreQuartiles"` // method 指定時のみ非nil
}

package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// DailyStockPickOutcome 答え合わせの勝敗判定。
type DailyStockPickOutcome string

const (
	DailyStockPickOutcomeWin  DailyStockPickOutcome = "win"
	DailyStockPickOutcomeLose DailyStockPickOutcome = "lose"
	DailyStockPickOutcomeDraw DailyStockPickOutcome = "draw"
	DailyStockPickOutcomeVoid DailyStockPickOutcome = "void" // 分割/併合・上場廃止・期限切れで公正に判定できない
)

// DailyStockPick daily_stock_pick のドメインモデル。答え合わせ前は Return*/Outcome/EvaluatedAt は nil。
type DailyStockPick struct {
	PickDate          time.Time
	StockBrandID      string
	TickerSymbol      string
	Name              string // DB非保存。Slack表示用に usecase 側で StockBrandRepository から解決する
	PickRank          int
	Score             decimal.Decimal
	ScoreVersion      string
	SignalCount       int
	Strategies        []string // DBへはカンマ区切りで保存
	Sector33CodeName  string
	BaseClosePrice    decimal.Decimal
	BaseAdjClosePrice decimal.Decimal
	AvgTradingValue   decimal.Decimal
	VolumeRatio       decimal.Decimal
	ADX               decimal.Decimal
	PlusDI            decimal.Decimal
	MinusDI           decimal.Decimal
	ATRRatio          decimal.Decimal
	RSI               decimal.Decimal
	Return1D          *decimal.Decimal
	Return3D          *decimal.Decimal
	Return5D          *decimal.Decimal
	Outcome           *DailyStockPickOutcome
	EvaluatedAt       *time.Time
	NotifiedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Evaluated 答え合わせ済みか。
func (p *DailyStockPick) Evaluated() bool {
	return p.EvaluatedAt != nil
}

// Notified Slack通知済みか。
func (p *DailyStockPick) Notified() bool {
	return p.NotifiedAt != nil
}

// DailyStockPickSummary 1 pick_date 分の答え合わせサマリ（Slack通知用）。
type DailyStockPickSummary struct {
	PickDate    time.Time
	Total       int
	Win         int
	Lose        int
	Draw        int
	Void        int
	WinRate     decimal.Decimal
	AvgReturn1D decimal.Decimal
	AvgReturn5D decimal.Decimal
	Best        *DailyStockPick
	Worst       *DailyStockPick
}

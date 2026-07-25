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

// --- API レスポンス用の構造体 ---

// DailyStockPickStrategy 点灯した戦略（キーと日本語表示名）。
type DailyStockPickStrategy struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DailyStockPickStatSummary 推奨群の集計。日別・スコア帯別・全体で同じ構造を使う。
// WinRate の分母は win+lose+draw（void 除外）、平均リターンの分母は各リターンが非nilの件数。
type DailyStockPickStatSummary struct {
	Total          int             `json:"total"`
	EvaluatedCount int             `json:"evaluatedCount"`
	PendingCount   int             `json:"pendingCount"`
	Win            int             `json:"win"`
	Lose           int             `json:"lose"`
	Draw           int             `json:"draw"`
	Void           int             `json:"void"`
	WinRate        decimal.Decimal `json:"winRate"`
	AvgReturn1D    decimal.Decimal `json:"avgReturn1d"`
	AvgReturn3D    decimal.Decimal `json:"avgReturn3d"`
	AvgReturn5D    decimal.Decimal `json:"avgReturn5d"`
}

// DailyStockPickItem API レスポンス用の推奨銘柄1件。答え合わせ前は Return*/Outcome/EvaluatedAt が null。
type DailyStockPickItem struct {
	PickRank         int                       `json:"pickRank"`
	StockBrandID     string                    `json:"stockBrandId"`
	TickerSymbol     string                    `json:"tickerSymbol"`
	Name             string                    `json:"name"`
	Score            decimal.Decimal           `json:"score"`
	ScoreVersion     string                    `json:"scoreVersion"`
	SignalCount      int                       `json:"signalCount"`
	Strategies       []*DailyStockPickStrategy `json:"strategies"`
	Sector33CodeName string                    `json:"sector33CodeName"`
	BaseClosePrice   decimal.Decimal           `json:"baseClosePrice"`
	AvgTradingValue  decimal.Decimal           `json:"avgTradingValue"`
	VolumeRatio      decimal.Decimal           `json:"volumeRatio"`
	ADX              decimal.Decimal           `json:"adx"`
	PlusDI           decimal.Decimal           `json:"plusDi"`
	MinusDI          decimal.Decimal           `json:"minusDi"`
	ATRRatio         decimal.Decimal           `json:"atrRatio"`
	RSI              decimal.Decimal           `json:"rsi"`
	Return1D         *decimal.Decimal          `json:"return1d"`
	Return3D         *decimal.Decimal          `json:"return3d"`
	Return5D         *decimal.Decimal          `json:"return5d"`
	Outcome          *DailyStockPickOutcome    `json:"outcome"`
	EvaluatedAt      *string                   `json:"evaluatedAt"`
}

// DailyStockPickDay GET /daily-stock-picks のレスポンス。
// 該当日のデータが無い場合は PickDate が null、Items が空配列になる（エラーにはしない）。
type DailyStockPickDay struct {
	PickDate     *string                   `json:"pickDate"`
	ScoreVersion string                    `json:"scoreVersion"`
	Evaluated    bool                      `json:"evaluated"`
	NotifiedAt   *string                   `json:"notifiedAt"`
	Summary      DailyStockPickStatSummary `json:"summary"`
	Items        []*DailyStockPickItem     `json:"items"`
}

// DailyStockPickDates GET /daily-stock-picks/dates のレスポンス（新しい順）。
type DailyStockPickDates struct {
	Dates []string `json:"dates"`
}

// DailyStockPickDailyStat 日別の成績（pick_date 昇順で返す）。
type DailyStockPickDailyStat struct {
	PickDate string `json:"pickDate"`
	DailyStockPickStatSummary
}

// DailyStockPickScoreBand スコア帯別の成績。Band は "70-79" 形式の表示用ラベル。
type DailyStockPickScoreBand struct {
	Band  string          `json:"band"`
	Lower decimal.Decimal `json:"lower"`
	Upper decimal.Decimal `json:"upper"`
	DailyStockPickStatSummary
}

// DailyStockPickStats GET /daily-stock-picks/stats のレスポンス。
type DailyStockPickStats struct {
	From         *string                    `json:"from"`
	To           *string                    `json:"to"`
	ScoreVersion string                     `json:"scoreVersion"`
	Totals       DailyStockPickStatSummary  `json:"totals"`
	Daily        []*DailyStockPickDailyStat `json:"daily"`
	ScoreBands   []*DailyStockPickScoreBand `json:"scoreBands"`
}

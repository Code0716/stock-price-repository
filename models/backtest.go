package models

import "github.com/shopspring/decimal"

// BacktestParams バックテストの共通イグジット・約定パラメータ。
type BacktestParams struct {
	TakeProfit     decimal.Decimal `json:"takeProfit"`     // 利確率（例: 0.10 = +10%）
	StopLoss       decimal.Decimal `json:"stopLoss"`       // 損切り率（例: 0.05 = -5%）
	MaxHoldDays    int             `json:"maxHoldDays"`    // 最大保有営業日数
	CommissionRate decimal.Decimal `json:"commissionRate"` // 片道手数料率（例: 0.0005）。ゼロ値 = コストなし
	SlippageRate   decimal.Decimal `json:"slippageRate"`   // 片道スリッページ率（例: 0.001）。ゼロ値 = コストなし
}

// BacktestEquityPoint エクイティカーブの1点（初期資金=1.0 を起点とした倍率）。
type BacktestEquityPoint struct {
	Date   string          `json:"date"`
	Equity decimal.Decimal `json:"equity"`
}

// BacktestTrade 1回の売買（エントリー〜手仕舞い）。
type BacktestTrade struct {
	EntryDate  string          `json:"entryDate"`
	ExitDate   string          `json:"exitDate"`
	EntryPrice decimal.Decimal `json:"entryPrice"`
	ExitPrice  decimal.Decimal `json:"exitPrice"`
	Return     decimal.Decimal `json:"return"` // (exit/entry - 1)
	HoldDays   int             `json:"holdDays"`
	// Reason: take_profit / stop_loss / max_hold / end_of_data
	Reason string `json:"reason"`
}

// BacktestResult 1戦略のバックテスト成績。
type BacktestResult struct {
	TotalReturn  decimal.Decimal `json:"totalReturn"`  // 期間トータルリターン（複利）
	Trades       int             `json:"trades"`       // 取引回数
	WinRate      decimal.Decimal `json:"winRate"`      // 勝率
	ProfitFactor decimal.Decimal `json:"profitFactor"` // 総利益/総損失（損失0なら0扱い）
	MaxDrawdown  decimal.Decimal `json:"maxDrawdown"`  // エクイティの最大下落率（負値）
	AvgWin       decimal.Decimal `json:"avgWin"`       // 平均利益（勝ちトレード）
	AvgLoss      decimal.Decimal `json:"avgLoss"`      // 平均損失（負値）
	PayoffRatio  decimal.Decimal `json:"payoffRatio"`  // |平均利益/平均損失|
	AvgHoldDays  float64         `json:"avgHoldDays"`
	Equity       []BacktestEquityPoint `json:"equity"`
	TradeList    []BacktestTrade       `json:"tradeList"`
}

// StrategyBacktest 戦略名つきの成績（ランキング表示用）。
type StrategyBacktest struct {
	Strategy string         `json:"strategy"` // 識別子
	Label    string         `json:"label"`    // 日本語表示名
	Result   BacktestResult `json:"result"`
}

// BacktestComparison 指定銘柄・期間における全戦略の比較結果（ランキング順）。
type BacktestComparison struct {
	Symbol      string             `json:"symbol"`
	From        string             `json:"from"`
	To          string             `json:"to"`
	TradingDays int                `json:"tradingDays"`
	Params      BacktestParams     `json:"params"`
	Strategies  []StrategyBacktest `json:"strategies"`
}

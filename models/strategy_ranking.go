package models

import "github.com/shopspring/decimal"

// StrategyRankingItem 1戦略の全銘柄横断集計。
type StrategyRankingItem struct {
	Strategy        string          `json:"strategy"`
	Label           string          `json:"label"`
	StockCount      int             `json:"stockCount"`      // 検証できた銘柄数（>=minBacktestDays）
	TradedStocks    int             `json:"tradedStocks"`    // 取引が1回以上発生した銘柄数
	AvgTotalReturn  decimal.Decimal `json:"avgTotalReturn"`  // 全検証銘柄平均（0取引=0%含む）
	PositiveRate    decimal.Decimal `json:"positiveRate"`    // totalReturn>0 の銘柄割合
	AvgWinRate      decimal.Decimal `json:"avgWinRate"`      // 取引のある銘柄での平均
	AvgProfitFactor decimal.Decimal `json:"avgProfitFactor"` // 取引のある銘柄での平均
	TotalTrades     int             `json:"totalTrades"`
	BestCount       int             `json:"bestCount"` // その銘柄で最高totalReturnだった戦略として選ばれた回数
}

// StrategyRanking 全戦略の横断集計（Redis に JSON で保存する）。
type StrategyRanking struct {
	Computed    bool                  `json:"computed"`    // バッチ未実行なら false
	ComputedAt  string                `json:"computedAt"`  // RFC3339 or ""
	Universe    string                `json:"universe"`    // "main_markets"
	TotalStocks int                   `json:"totalStocks"` // ユニバースの銘柄数
	Params      BacktestParams        `json:"params"`
	Items       []StrategyRankingItem `json:"items"` // AvgTotalReturn 降順
}

// StrategyStockResult 1戦略×1銘柄のバックテスト結果（ドリルダウン用）。
type StrategyStockResult struct {
	TickerSymbol string          `json:"tickerSymbol"`
	Name         string          `json:"name"`
	TotalReturn  decimal.Decimal `json:"totalReturn"`
	Trades       int             `json:"trades"`
	WinRate      decimal.Decimal `json:"winRate"`
	ProfitFactor decimal.Decimal `json:"profitFactor"`
	MaxDrawdown  decimal.Decimal `json:"maxDrawdown"`
	PayoffRatio  decimal.Decimal `json:"payoffRatio"`
	AvgHoldDays  float64         `json:"avgHoldDays"`
}

// StrategyStocks ドリルダウン API レスポンス（Redis に JSON で保存する）。
type StrategyStocks struct {
	Computed   bool                   `json:"computed"`   // バッチ未実行なら false
	ComputedAt string                 `json:"computedAt"` // RFC3339 or ""
	Strategy   string                 `json:"strategy"`
	Label      string                 `json:"label"`
	TotalCount int                    `json:"totalCount"` // limit 適用前の全件数
	Items      []*StrategyStockResult `json:"items"`      // TotalReturn 降順
}

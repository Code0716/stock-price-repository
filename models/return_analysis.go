package models

import "github.com/shopspring/decimal"

// BenchmarkNikkei リターン分析のベンチマーク識別子（日経平均）。
const BenchmarkNikkei = "nikkei"

// ReturnAnalysis 指定銘柄の期間リターン・リスク指標・対ベンチマーク指標をまとめた分析結果。
// 価格はいずれも調整後終値（adjClose）ベースで算出される。
type ReturnAnalysis struct {
	Symbol    string `json:"symbol"`
	Benchmark string `json:"benchmark"`
	// From / To は実際にデータが存在した範囲（YYYY-MM-DD）。リクエスト値ではなく結合後の実データ範囲。
	From string `json:"from"`
	To   string `json:"to"`
	// TradingDays 分析に用いた営業日数（銘柄とベンチマークの両方が存在した日数）。
	TradingDays int `json:"tradingDays"`

	CumulativeReturn     decimal.Decimal `json:"cumulativeReturn"`
	AnnualizedReturn     decimal.Decimal `json:"annualizedReturn"`
	AnnualizedVolatility decimal.Decimal `json:"annualizedVolatility"`
	// MaxDrawdown ピークからの最大下落率（負の小数。例: -0.23）。
	MaxDrawdown  decimal.Decimal `json:"maxDrawdown"`
	SharpeRatio  decimal.Decimal `json:"sharpeRatio"`
	SortinoRatio decimal.Decimal `json:"sortinoRatio"`
	CalmarRatio  decimal.Decimal `json:"calmarRatio"`
	// Beta / Correlation はベンチマーク（日経平均）に対する値。
	Beta        decimal.Decimal `json:"beta"`
	Correlation decimal.Decimal `json:"correlation"`
	// BenchmarkReturn ベンチマークの同期間累積リターン。
	BenchmarkReturn decimal.Decimal `json:"benchmarkReturn"`
	// ExcessReturn 銘柄累積リターン − ベンチマーク累積リターン（相対力）。
	ExcessReturn decimal.Decimal `json:"excessReturn"`
}

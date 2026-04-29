package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// AppliedStockConsolidationHistory は株式併合の適用履歴を表すドメインモデルです。
type AppliedStockConsolidationHistory struct {
	ID                uint64
	Symbol            string
	ConsolidationDate time.Time
	Ratio             decimal.Decimal
	AppliedAt         time.Time
}

// NewAppliedStockConsolidationHistory は新しい AppliedStockConsolidationHistory インスタンスを作成します。
// ratio は「旧株数 / 新株数」を表す（例: 5株を1株に併合する場合は 5）。
func NewAppliedStockConsolidationHistory(symbol string, consolidationDate time.Time, ratio decimal.Decimal) *AppliedStockConsolidationHistory {
	return &AppliedStockConsolidationHistory{
		Symbol:            symbol,
		ConsolidationDate: consolidationDate,
		Ratio:             ratio,
		AppliedAt:         time.Now(),
	}
}

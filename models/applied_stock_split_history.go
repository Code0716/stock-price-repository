package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// AppliedStockSplitHistory は株式分割の適用履歴を表すドメインモデルです。
type AppliedStockSplitHistory struct {
	ID        uint64
	Symbol    string
	SplitDate time.Time
	Ratio     decimal.Decimal
	AppliedAt time.Time
}

// NewAppliedStockSplitHistory は新しい AppliedStockSplitHistory インスタンスを作成します。
func NewAppliedStockSplitHistory(symbol string, splitDate time.Time, ratio decimal.Decimal) *AppliedStockSplitHistory {
	return &AppliedStockSplitHistory{
		Symbol:    symbol,
		SplitDate: splitDate,
		Ratio:     ratio,
		AppliedAt: time.Now(),
	}
}

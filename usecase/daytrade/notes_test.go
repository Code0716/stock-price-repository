package daytrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

var noteBaseDate = time.Date(2026, 5, 21, 0, 0, 0, 0, time.Local)

func TestMergeTradesWithNotes(t *testing.T) {
	trades := []*models.DaytradeTradeApprox{
		{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: noteBaseDate, Direction: "返済売", ProfitLoss: -5000, TradeAmount: 500000},
		{TickerSymbol: "5803", BrandName: "FujiKumi", ExecutedOn: noteBaseDate, Direction: "返済売", ProfitLoss: 2000, TradeAmount: 200000},
		{TickerSymbol: "7974", BrandName: "Nintendo", ExecutedOn: noteBaseDate, Direction: "返済売", ProfitLoss: 1000, TradeAmount: 100000},
	}

	tests := []struct {
		name         string
		notes        []*models.DaytradeTradeNoteRecord
		wantNoteKeys []string // tickerSymbol of trades that should have note (nil = no note)
	}{
		{
			name:         "注釈なし → Note が nil",
			notes:        []*models.DaytradeTradeNoteRecord{},
			wantNoteKeys: []string{"", "", ""},
		},
		{
			name: "1件に注釈あり → 該当行のみ Note が非nil",
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: noteBaseDate, Direction: "返済売", Tags: []string{"高値掴み"}},
			},
			wantNoteKeys: []string{"9984", "", ""},
		},
		{
			name: "再インポート相当: 同一自然キーの注釈は保持される",
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "5803", ExecutedOn: noteBaseDate, Direction: "返済売", Memo: "反省メモ"},
			},
			wantNoteKeys: []string{"", "5803", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeTradesWithNotes(trades, tt.notes)
			assert.Len(t, got, len(trades))
			for i, sym := range tt.wantNoteKeys {
				if sym == "" {
					assert.Nil(t, got[i].Note, "trade[%d] Note should be nil", i)
				} else {
					assert.NotNil(t, got[i].Note, "trade[%d] Note should not be nil", i)
					assert.Equal(t, sym, got[i].TickerSymbol)
				}
			}
		})
	}
}

func TestComputeTagStats(t *testing.T) {
	withNote := func(pnl int64, tags []string) *models.DaytradeTradeWithNote {
		return &models.DaytradeTradeWithNote{
			ProfitLoss: pnl,
			Note:       &models.DaytradeTradeNote{Tags: tags},
		}
	}

	tests := []struct {
		name       string
		trades     []*models.DaytradeTradeWithNote
		wantTags   []string  // 期待するタグの順序（TotalPnl昇順）
		wantPnls   []int64
		wantCounts []int
	}{
		{
			name:       "注釈なし → 空リスト",
			trades:     []*models.DaytradeTradeWithNote{{ProfitLoss: -1000}},
			wantTags:   []string{},
			wantPnls:   []int64{},
			wantCounts: []int{},
		},
		{
			name: "複数タグが独立集計される",
			trades: []*models.DaytradeTradeWithNote{
				withNote(-3000, []string{"高値掴み", "損切り遅れ"}),
				withNote(-1000, []string{"高値掴み"}),
				withNote(2000, []string{"順張り"}),
			},
			wantTags:   []string{"高値掴み", "損切り遅れ", "順張り"}, // 高値掴み=-4000, 損切り遅れ=-3000, 順張り=2000
			wantPnls:   []int64{-4000, -3000, 2000},
			wantCounts: []int{2, 1, 1},
		},
		{
			name: "タグなしトレードは無視される",
			trades: []*models.DaytradeTradeWithNote{
				{ProfitLoss: -5000},
				withNote(-1000, []string{"ナンピン"}),
			},
			wantTags:   []string{"ナンピン"},
			wantPnls:   []int64{-1000},
			wantCounts: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTagStats(tt.trades)
			assert.Len(t, got, len(tt.wantTags))
			for i, tag := range tt.wantTags {
				if i < len(got) {
					assert.Equal(t, tag, got[i].Tag)
					assert.Equal(t, tt.wantPnls[i], got[i].TotalPnl)
					assert.Equal(t, tt.wantCounts[i], got[i].TradeCount)
				}
			}
		})
	}
}

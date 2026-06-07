package daytrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

var baseDate = time.Date(2026, 5, 21, 0, 0, 0, 0, time.Local)

func TestNormalizeDirection(t *testing.T) {
	tests := []struct {
		name       string
		tradeKind  string
		marginKind string
		want       string
	}{
		{"新フォーマット: tradeKind空 → marginKindを使用", "", "返済売", "返済売"},
		{"新フォーマット: tradeKind空 → marginKind返済買", "", "返済買", "返済買"},
		{"旧フォーマット: tradeKind優先", "売建", "返済買", "売建"},
		{"旧フォーマット: 現物売却", "現物売却", "", "現物売却"},
		{"両方空", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDirection(tt.tradeKind, tt.marginKind)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildTradeApprox(t *testing.T) {
	tests := []struct {
		name       string
		executions []*models.DaytradeExecution
		wantLen    int
		wantPnl    map[string]int64 // key: ticker+direction
	}{
		{
			name: "分割決済2行が1トレードに集約される",
			executions: []*models.DaytradeExecution{
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, MarginKind: "返済売", ProfitLoss: 1000, TradeAmount: 500000},
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, MarginKind: "返済売", ProfitLoss: 500, TradeAmount: 300000},
			},
			wantLen: 1,
			wantPnl: map[string]int64{"9984:返済売": 1500},
		},
		{
			name: "同一銘柄・同日・別方向は別トレード",
			executions: []*models.DaytradeExecution{
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, MarginKind: "返済売", ProfitLoss: 1000, TradeAmount: 500000},
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, MarginKind: "返済買", ProfitLoss: -500, TradeAmount: 300000},
			},
			wantLen: 2,
			wantPnl: map[string]int64{"9984:返済売": 1000, "9984:返済買": -500},
		},
		{
			name: "別銘柄は別トレード",
			executions: []*models.DaytradeExecution{
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, MarginKind: "返済売", ProfitLoss: 1000, TradeAmount: 500000},
				{TickerSymbol: "5803", BrandName: "FujiKumi", ExecutedOn: baseDate, MarginKind: "返済売", ProfitLoss: -800, TradeAmount: 200000},
			},
			wantLen: 2,
			wantPnl: map[string]int64{"9984:返済売": 1000, "5803:返済売": -800},
		},
		{
			name: "旧フォーマット(tradeKind非空)でも正しく集約",
			executions: []*models.DaytradeExecution{
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, TradeKind: "売建", MarginKind: "返済売", ProfitLoss: 1000, TradeAmount: 500000},
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, TradeKind: "売建", MarginKind: "返済売", ProfitLoss: 460, TradeAmount: 300000},
			},
			wantLen: 1,
			wantPnl: map[string]int64{"9984:売建": 1460},
		},
		{
			name:       "空スライス",
			executions: []*models.DaytradeExecution{},
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildTradeApprox(tt.executions)
			assert.Len(t, got, tt.wantLen)
			for _, trade := range got {
				key := trade.TickerSymbol + ":" + trade.Direction
				if expected, ok := tt.wantPnl[key]; ok {
					assert.Equal(t, expected, trade.ProfitLoss, "ProfitLoss for %s", key)
				}
			}
		})
	}
}

func TestComputeLossConcentration(t *testing.T) {
	tests := []struct {
		name          string
		trades        []*models.DaytradeTradeApprox
		wantTotalLoss int64
		wantTop1Min   float64 // top1 ratio >= this
		wantTop1Max   float64 // top1 ratio <= this
		wantWorstLen  int
	}{
		{
			name: "損失3件: 上位1件の寄与率が最大",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "A", BrandName: "A", ExecutedOn: baseDate, Direction: "返済売", ProfitLoss: -10000},
				{TickerSymbol: "B", BrandName: "B", ExecutedOn: baseDate, Direction: "返済売", ProfitLoss: -3000},
				{TickerSymbol: "C", BrandName: "C", ExecutedOn: baseDate, Direction: "返済売", ProfitLoss: -1000},
				{TickerSymbol: "D", BrandName: "D", ExecutedOn: baseDate, Direction: "返済売", ProfitLoss: 2000}, // 勝ちは含めない
			},
			wantTotalLoss: 14000,
			wantTop1Min:   0.71,
			wantTop1Max:   0.72,
			wantWorstLen:  3,
		},
		{
			name: "損失ゼロ（全勝）",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "A", BrandName: "A", ExecutedOn: baseDate, Direction: "返済売", ProfitLoss: 5000},
			},
			wantTotalLoss: 0,
			wantTop1Min:   0,
			wantTop1Max:   0,
			wantWorstLen:  0,
		},
		{
			name:          "空スライス",
			trades:        []*models.DaytradeTradeApprox{},
			wantTotalLoss: 0,
			wantTop1Min:   0,
			wantTop1Max:   0,
			wantWorstLen:  0,
		},
		{
			name: "損失が5件以上 → worst は5件まで",
			trades: func() []*models.DaytradeTradeApprox {
				trades := make([]*models.DaytradeTradeApprox, 7)
				for i := range trades {
					trades[i] = &models.DaytradeTradeApprox{
						TickerSymbol: "X",
						BrandName:    "X",
						ExecutedOn:   baseDate,
						Direction:    "返済売",
						ProfitLoss:   int64(-(i + 1) * 1000),
					}
				}
				return trades
			}(),
			wantTotalLoss: 28000,
			wantTop1Min:   0.24,
			wantTop1Max:   0.26,
			wantWorstLen:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeLossConcentration(tt.trades)
			assert.Equal(t, tt.wantTotalLoss, got.TotalLoss)
			assert.GreaterOrEqual(t, got.Top1Ratio, tt.wantTop1Min)
			assert.LessOrEqual(t, got.Top1Ratio, tt.wantTop1Max)
			assert.Len(t, got.WorstTrades, tt.wantWorstLen)
			// Top3Ratio >= Top1Ratio
			assert.GreaterOrEqual(t, got.Top3Ratio, got.Top1Ratio)
			// Top5Ratio >= Top3Ratio
			assert.GreaterOrEqual(t, got.Top5Ratio, got.Top3Ratio)
		})
	}
}

func TestComputeFavoriteTraps(t *testing.T) {
	tests := []struct {
		name      string
		trades    []*models.DaytradeTradeApprox
		wantSyms  []string  // 期待される ticker の順序（回数降順）
		wantCount []int
	}{
		{
			name: "期待値プラスは除外、マイナスだけ回数降順",
			trades: []*models.DaytradeTradeApprox{
				// 9984: 3回取引、合計マイナス
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, ProfitLoss: 1000},
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate.AddDate(0, 0, 1), ProfitLoss: -2000},
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate.AddDate(0, 0, 2), ProfitLoss: -500},
				// 5803: 2回取引、合計マイナス
				{TickerSymbol: "5803", BrandName: "FujiKumi", ExecutedOn: baseDate, ProfitLoss: -800},
				{TickerSymbol: "5803", BrandName: "FujiKumi", ExecutedOn: baseDate.AddDate(0, 0, 1), ProfitLoss: -200},
				// 7974: 1回取引、合計プラス（除外される）
				{TickerSymbol: "7974", BrandName: "Nintendo", ExecutedOn: baseDate, ProfitLoss: 3000},
			},
			wantSyms:  []string{"9984", "5803"},
			wantCount: []int{3, 2},
		},
		{
			name: "全銘柄がプラス → 空リスト",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", BrandName: "SBG", ExecutedOn: baseDate, ProfitLoss: 1000},
			},
			wantSyms:  []string{},
			wantCount: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeFavoriteTraps(tt.trades)
			assert.Len(t, got, len(tt.wantSyms))
			for i, sym := range tt.wantSyms {
				if i < len(got) {
					assert.Equal(t, sym, got[i].TickerSymbol)
					assert.Equal(t, tt.wantCount[i], got[i].TradeCount)
					assert.Less(t, got[i].TotalPnl, int64(0), "惚れ込み銘柄のPnLは負のはず")
				}
			}
		})
	}
}

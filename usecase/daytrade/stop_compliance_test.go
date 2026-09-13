package daytrade

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestComputeStopCompliance(t *testing.T) {
	stop := d("100")
	tests := []struct {
		name       string
		trades     []*models.DaytradeTradeApprox
		notes      []*models.DaytradeTradeNoteRecord
		priceByKey map[DailyLowHighKey]DailyLowHigh
		want       *models.DaytradeStopCompliance
	}{
		{
			name:   "宣言なしトレードは対象外",
			trades: []*models.DaytradeTradeApprox{{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: -500}},
			notes:  nil,
			want:   &models.DaytradeStopCompliance{},
		},
		{
			name: "ロング: 安値が宣言ストップに未到達 → not_triggered",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: 300},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "9984", ExecutedOn: "2026-05-21"}: {Low: d("105"), High: d("115")},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, NotTriggeredCount: 1},
		},
		{
			name: "ロング: 到達し宣言相当の損失で手仕舞い → honored",
			trades: []*models.DaytradeTradeApprox{
				// 取得単価110、ストップ100 → 想定損失 = (110-100)*100 = 1000。実損失900 <= 1000 なので honored
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: -900},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "9984", ExecutedOn: "2026-05-21"}: {Low: d("98"), High: d("112")},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, HonoredCount: 1},
		},
		{
			name: "ロング: 到達したが持ち越して黒字化 → breached_recovered",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: 200},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "9984", ExecutedOn: "2026-05-21"}: {Low: d("95"), High: d("125")},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, BreachedRecoveredCount: 1},
		},
		{
			name: "ロング: 到達し持ち越した結果、宣言より悪化 → breached_worse で上振れ額を集計",
			trades: []*models.DaytradeTradeApprox{
				// 想定損失=(110-100)*100=1000。実損失2000 → 上振れ1000
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: -2000},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "9984", ExecutedOn: "2026-05-21"}: {Low: d("90"), High: d("112")},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, BreachedWorseCount: 1, TotalOvershoot: 1000, AvgOvershoot: 1000},
		},
		{
			name: "ショート: 高値が宣言ストップ以上 → 到達判定される",
			trades: []*models.DaytradeTradeApprox{
				// 取得単価90（空売り建値）、ストップ100（建値より高い＝ショート想定）
				{TickerSymbol: "5803", ExecutedOn: baseDate, Direction: "信用新規売", Quantity: 100, AvgAcquisitionPrice: d("90"), ProfitLoss: -500},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "5803", ExecutedOn: baseDate, Direction: "信用新規売", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "5803", ExecutedOn: "2026-05-21"}: {Low: d("85"), High: d("105")},
			},
			// 想定損失=(100-90)*100=1000, 実損失500 <= 1000 → honored
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, HonoredCount: 1},
		},
		{
			name: "数量ゼロは unknown",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 0, AvgAcquisitionPrice: decimal.Zero, ProfitLoss: -500},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, UnknownCount: 1},
		},
		{
			name: "宣言価格が取得単価と一致は方向不明で unknown",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("100"), ProfitLoss: -500},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: map[DailyLowHighKey]DailyLowHigh{
				{TickerSymbol: "9984", ExecutedOn: "2026-05-21"}: {Low: d("90"), High: d("110")},
			},
			want: &models.DaytradeStopCompliance{DeclaredCount: 1, UnknownCount: 1},
		},
		{
			name: "当日の日足が無い場合は unknown",
			trades: []*models.DaytradeTradeApprox{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", Quantity: 100, AvgAcquisitionPrice: d("110"), ProfitLoss: -500},
			},
			notes: []*models.DaytradeTradeNoteRecord{
				{TickerSymbol: "9984", ExecutedOn: baseDate, Direction: "現物買", DeclaredStopPrice: &stop},
			},
			priceByKey: nil,
			want:       &models.DaytradeStopCompliance{DeclaredCount: 1, UnknownCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeStopCompliance(tt.trades, tt.notes, tt.priceByKey)
			assert.Equal(t, tt.want.DeclaredCount, got.DeclaredCount)
			assert.Equal(t, tt.want.NotTriggeredCount, got.NotTriggeredCount)
			assert.Equal(t, tt.want.HonoredCount, got.HonoredCount)
			assert.Equal(t, tt.want.BreachedRecoveredCount, got.BreachedRecoveredCount)
			assert.Equal(t, tt.want.BreachedWorseCount, got.BreachedWorseCount)
			assert.Equal(t, tt.want.UnknownCount, got.UnknownCount)
			assert.Equal(t, tt.want.TotalOvershoot, got.TotalOvershoot)
			assert.Equal(t, tt.want.AvgOvershoot, got.AvgOvershoot)
		})
	}
}

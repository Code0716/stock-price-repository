package daytrade

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func execution(marginKind string, quantity uint32, unitPrice, averageCost string, profitLoss int64) *models.DaytradeExecution {
	return &models.DaytradeExecution{
		MarginKind:  marginKind,
		Quantity:    quantity,
		UnitPrice:   dec(unitPrice),
		AverageCost: dec(averageCost),
		ProfitLoss:  profitLoss,
	}
}

func TestExecutionFee(t *testing.T) {
	tests := []struct {
		name      string
		ex        *models.DaytradeExecution
		wantGross int64
		wantFee   int64
	}{
		{
			name:      "ロング決済（返済売）手数料あり",
			ex:        execution("返済売", 100, "1000", "990", 995),
			wantGross: 1000,
			wantFee:   -5,
		},
		{
			name:      "ショート決済（返済買）手数料あり",
			ex:        execution("返済買", 100, "990", "1000", 997),
			wantGross: 1000,
			wantFee:   -3,
		},
		{
			name:      "日計り信用は手数料0",
			ex:        execution("返済売", 100, "1000", "990", 1000),
			wantGross: 1000,
			wantFee:   0,
		},
		{
			name:      "丸め誤差 fee=-1 は0扱い",
			ex:        execution("返済売", 100, "1000", "990", 999),
			wantGross: 1000,
			wantFee:   0,
		},
		{
			name:      "丸め誤差 fee=+1 は0扱い",
			ex:        execution("返済売", 100, "1000", "990", 1001),
			wantGross: 1000,
			wantFee:   0,
		},
		{
			name:      "fee>1 は異常値としてそのまま返す",
			ex:        execution("返済売", 100, "1000", "990", 1010),
			wantGross: 1000,
			wantFee:   10,
		},
		{
			name:      "小数点単価は1円単位に四捨五入してからfeeを算出",
			ex:        execution("返済売", 200, "5944.3", "5937", 1460),
			wantGross: 1460, // (5944.3-5937)*200=1460
			wantFee:   0,
		},
		{
			name:      "小数点単価・ショート決済",
			ex:        execution("返済買", 100, "5399.6", "5420", 1932),
			wantGross: 2040, // (5420-5399.6)*100=2040
			wantFee:   -108,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gross, fee := ExecutionFee(tt.ex)
			assert.Equal(t, tt.wantGross, gross)
			assert.Equal(t, tt.wantFee, fee)
		})
	}
}

func TestIsFeeAnomaly(t *testing.T) {
	assert.False(t, IsFeeAnomaly(0))
	assert.False(t, IsFeeAnomaly(-100))
	assert.False(t, IsFeeAnomaly(1))
	assert.True(t, IsFeeAnomaly(2))
	assert.True(t, IsFeeAnomaly(100))
}

func TestApplyFee(t *testing.T) {
	ex := execution("返済売", 100, "1000", "990", 995)
	ApplyFee(ex)
	assert.Equal(t, int64(-5), ex.Fee)
	assert.Equal(t, int64(1000), ex.GrossProfitLoss)
}

func mustDate(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestFeeByBucket(t *testing.T) {
	executions := []*models.DaytradeExecution{
		{ExecutedOn: mustDate("2026-09-01"), TickerSymbol: "6981", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 995}, // fee=-5
		{ExecutedOn: mustDate("2026-09-01"), TickerSymbol: "6981", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 1000}, // fee=0
		{ExecutedOn: mustDate("2026-09-25"), TickerSymbol: "5801", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 992}, // fee=-8
		{ExecutedOn: mustDate("2026-10-01"), TickerSymbol: "5801", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 985}, // fee=-15
	}

	t.Run("daily", func(t *testing.T) {
		got := FeeByBucket(executions, models.DaytradeSummaryGranularityDaily)
		assert.Equal(t, int64(-5), got["2026-09-01"])
		assert.Equal(t, int64(-8), got["2026-09-25"])
		assert.Equal(t, int64(-15), got["2026-10-01"])
	})

	t.Run("monthly", func(t *testing.T) {
		got := FeeByBucket(executions, models.DaytradeSummaryGranularityMonthly)
		assert.Equal(t, int64(-13), got["2026-09-01"])
		assert.Equal(t, int64(-15), got["2026-10-01"])
	})

	t.Run("yearly", func(t *testing.T) {
		got := FeeByBucket(executions, models.DaytradeSummaryGranularityYearly)
		assert.Equal(t, int64(-28), got["2026-01-01"])
	})

	t.Run("all", func(t *testing.T) {
		got := FeeByBucket(executions, models.DaytradeSummaryGranularityAll)
		assert.Equal(t, int64(-28), got[""])
	})
}

func TestBuildFeeReport(t *testing.T) {
	executions := []*models.DaytradeExecution{
		{ID: 1, ExecutedOn: mustDate("2026-09-01"), TickerSymbol: "6981", BrandName: "村田製作所", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 995}, // fee=-5
		{ID: 2, ExecutedOn: mustDate("2026-09-01"), TickerSymbol: "6981", BrandName: "村田製作所", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 1000}, // fee=0
		{ID: 3, ExecutedOn: mustDate("2026-09-25"), TickerSymbol: "5801", BrandName: "テスト銘柄", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 992}, // fee=-8
		{ID: 4, ExecutedOn: mustDate("2026-09-25"), TickerSymbol: "5801", BrandName: "テスト銘柄", MarginKind: "返済売",
			Quantity: 100, UnitPrice: dec("1000"), AverageCost: dec("990"), ProfitLoss: 1010}, // fee=10 (異常)
	}

	report := BuildFeeReport(executions)

	assert.Equal(t, int64(-3), report.TotalFee) // -5 + -8 + 10（fee!=0の全合計。異常行も含む）
	assert.Equal(t, 3, report.FeeRowCount)      // fee!=0 の行数（-5, -8, +10）
	assert.Equal(t, 2, report.FeeDayCount)      // 9/1, 9/25 の2日
	assert.Len(t, report.Anomalies, 1)
	assert.Equal(t, uint64(4), report.Anomalies[0].ID)
	assert.Equal(t, int64(10), report.Anomalies[0].Fee)
	assert.Len(t, report.Monthly, 1)
	assert.Equal(t, "2026-09", report.Monthly[0].Month)
	assert.Equal(t, int64(-3), report.Monthly[0].Fee)
	assert.Equal(t, int64(3997), report.Monthly[0].ProfitLoss)
	assert.Equal(t, int64(4000), report.Monthly[0].GrossProfitLoss)
	assert.Equal(t, int64(3997), report.ProfitLoss)
	assert.Equal(t, int64(4000), report.GrossProfitLoss)
}

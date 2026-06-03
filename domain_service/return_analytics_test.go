package domain_service

import (
	"testing"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func decs(fs ...float64) []decimal.Decimal {
	out := make([]decimal.Decimal, 0, len(fs))
	for _, f := range fs {
		out = append(out, decimal.NewFromFloat(f))
	}
	return out
}

func f64(d decimal.Decimal) float64 {
	v, _ := d.Float64()
	return v
}

func TestExtractAdjClosePrices(t *testing.T) {
	prices := []*models.StockBrandDailyPrice{
		{Adjclose: decimal.NewFromFloat(100)},
		{Adjclose: decimal.NewFromFloat(110)},
	}
	got := ExtractAdjClosePrices(prices)
	assert.Len(t, got, 2)
	assert.InDelta(t, 100, f64(got[0]), 1e-9)
	assert.InDelta(t, 110, f64(got[1]), 1e-9)
	assert.Empty(t, ExtractAdjClosePrices(nil))
}

func TestDailyReturns(t *testing.T) {
	tests := []struct {
		name   string
		prices []decimal.Decimal
		want   []float64
	}{
		{"通常", decs(100, 110, 99), []float64{0.1, -0.1}},
		{"データ1点は nil", decs(100), nil},
		{"空は nil", nil, nil},
		{"直前ゼロは0扱い", decs(0, 100), []float64{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DailyReturns(tt.prices)
			assert.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.InDelta(t, tt.want[i], f64(got[i]), 1e-9)
			}
		})
	}
}

func TestCumulativeReturn(t *testing.T) {
	tests := []struct {
		name   string
		prices []decimal.Decimal
		want   float64
	}{
		{"通常", decs(100, 110, 99), -0.01},
		{"上昇", decs(100, 150), 0.5},
		{"1点は0", decs(100), 0},
		{"先頭ゼロは0", decs(0, 100), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, f64(CumulativeReturn(tt.prices)), 1e-9)
		})
	}
}

func TestAnnualizedReturn(t *testing.T) {
	tests := []struct {
		name             string
		cumulativeReturn decimal.Decimal
		numReturns       int
		want             float64
	}{
		{"1年=そのまま", decimal.NewFromFloat(0.1), 252, 0.1},
		{"半年=2乗", decimal.NewFromFloat(0.1), 126, 0.21},
		{"日数0は0", decimal.NewFromFloat(0.1), 0, 0},
		{"全損は-1", decimal.NewFromFloat(-1), 100, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, f64(AnnualizedReturn(tt.cumulativeReturn, tt.numReturns)), 1e-9)
		})
	}
}

func TestAnnualizedVolatility(t *testing.T) {
	// returns: 0.01,-0.01,0.02,-0.02 → mean0, sampleVar=0.001/3, std=0.0182574, *sqrt(252)=0.289828
	assert.InDelta(t, 0.289828, f64(AnnualizedVolatility(decs(0.01, -0.01, 0.02, -0.02))), 1e-5)
	assert.InDelta(t, 0, f64(AnnualizedVolatility(decs(0.01))), 1e-9)
}

func TestAnnualizedDownsideDeviation(t *testing.T) {
	// 負のみ: -0.01,-0.02 → sumsq=0.0005, /(n-1)=0.0001667, sqrt=0.0129099, *sqrt(252)=0.204939
	got := AnnualizedDownsideDeviation(decs(0.01, -0.01, 0.02, -0.02), decimal.Zero)
	assert.InDelta(t, 0.204939, f64(got), 1e-5)
	assert.InDelta(t, 0, f64(AnnualizedDownsideDeviation(decs(0.01), decimal.Zero)), 1e-9)
}

func TestMaxDrawdown(t *testing.T) {
	tests := []struct {
		name   string
		prices []decimal.Decimal
		want   float64
	}{
		{"通常", decs(100, 120, 90, 110), -0.25},
		{"単調増加はDDなし", decs(100, 110, 120), 0},
		{"空は0", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, f64(MaxDrawdown(tt.prices)), 1e-9)
		})
	}
}

func TestSharpeRatio(t *testing.T) {
	assert.InDelta(t, 2.0, f64(SharpeRatio(decimal.NewFromFloat(0.2), decimal.NewFromFloat(0.1), decimal.Zero)), 1e-9)
	assert.InDelta(t, 0, f64(SharpeRatio(decimal.NewFromFloat(0.2), decimal.Zero, decimal.Zero)), 1e-9)
}

func TestSortinoRatio(t *testing.T) {
	assert.InDelta(t, 2.0, f64(SortinoRatio(decimal.NewFromFloat(0.2), decimal.NewFromFloat(0.1), decimal.Zero)), 1e-9)
	assert.InDelta(t, 0, f64(SortinoRatio(decimal.NewFromFloat(0.2), decimal.Zero, decimal.Zero)), 1e-9)
}

func TestCalmarRatio(t *testing.T) {
	assert.InDelta(t, 2.0, f64(CalmarRatio(decimal.NewFromFloat(0.3), decimal.NewFromFloat(-0.15))), 1e-9)
	assert.InDelta(t, 0, f64(CalmarRatio(decimal.NewFromFloat(0.3), decimal.Zero)), 1e-9)
}

func TestBeta(t *testing.T) {
	bench := decs(0.01, -0.02, 0.03, 0.00)
	stock := decs(0.02, -0.04, 0.06, 0.00) // = 2 * bench
	assert.InDelta(t, 2.0, f64(Beta(stock, bench)), 1e-6)
	assert.InDelta(t, 0, f64(Beta(decs(0.01), bench)), 1e-9)              // 長さ不一致
	assert.InDelta(t, 0, f64(Beta(stock, decs(0.0, 0.0, 0.0, 0.0))), 1e-9) // ベンチ分散0
}

func TestCorrelation(t *testing.T) {
	bench := decs(0.01, -0.02, 0.03, 0.00)
	assert.InDelta(t, 1.0, f64(Correlation(decs(0.02, -0.04, 0.06, 0.00), bench)), 1e-6)  // 完全正相関
	assert.InDelta(t, -1.0, f64(Correlation(decs(-0.01, 0.02, -0.03, 0.00), bench)), 1e-6) // 完全負相関
	assert.InDelta(t, 0, f64(Correlation(decs(0.01), bench)), 1e-9)                          // 長さ不一致
}

func TestExcessReturn(t *testing.T) {
	assert.InDelta(t, 0.2, f64(ExcessReturn(decimal.NewFromFloat(0.3), decimal.NewFromFloat(0.1))), 1e-9)
}

func TestMean(t *testing.T) {
	assert.InDelta(t, 2.0, f64(mean(decs(1, 2, 3))), 1e-9)
	assert.InDelta(t, 0, f64(mean(nil)), 1e-9)
}

func TestSampleVariance(t *testing.T) {
	assert.InDelta(t, 1.0, f64(sampleVariance(decs(1, 2, 3))), 1e-9)
	assert.InDelta(t, 0, f64(sampleVariance(decs(5))), 1e-9)
}

func TestStdDevSample(t *testing.T) {
	assert.InDelta(t, 1.0, f64(stdDevSample(decs(1, 2, 3))), 1e-9)
}

func TestSampleCovariance(t *testing.T) {
	assert.InDelta(t, 2.0, f64(sampleCovariance(decs(1, 2, 3), decs(2, 4, 6))), 1e-9)
	assert.InDelta(t, 0, f64(sampleCovariance(decs(1, 2), decs(1, 2, 3))), 1e-9) // 長さ不一致
}

func TestDecimalSqrt(t *testing.T) {
	assert.InDelta(t, 2.0, f64(decimalSqrt(decimal.NewFromInt(4))), 1e-9)
	assert.InDelta(t, 0, f64(decimalSqrt(decimal.Zero)), 1e-9)
	assert.InDelta(t, 0, f64(decimalSqrt(decimal.NewFromInt(-1))), 1e-9)
}

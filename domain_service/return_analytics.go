// Package domain_service はドメインの純粋な計算ロジックを提供する。
// DB やネットワークなどの外部依存を持たず、入力スライスから指標を算出する。
package domain_service

import (
	"math"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
)

// tradingDaysPerYear 年率換算に用いる年間営業日数（日本株の慣例値）
const tradingDaysPerYear = 252

// ExtractAdjClosePrices 日足スライスから調整後終値（adjClose）のリストを抽出する。
// リターン計算は分割・配当調整済みの adjClose を基準にするのが正確。
func ExtractAdjClosePrices(prices []*models.StockBrandDailyPrice) []decimal.Decimal {
	out := make([]decimal.Decimal, 0, len(prices))
	for _, p := range prices {
		out = append(out, p.Adjclose)
	}
	return out
}

// DailyReturns 価格系列から日次単純リターン (p[i]/p[i-1] - 1) を算出する。
// 返り値の長さは len(prices)-1。直前値がゼロの場合はその日のリターンを 0 とする。
func DailyReturns(prices []decimal.Decimal) []decimal.Decimal {
	if len(prices) < 2 {
		return nil
	}
	out := make([]decimal.Decimal, 0, len(prices)-1)
	one := decimal.NewFromInt(1)
	for i := 1; i < len(prices); i++ {
		prev := prices[i-1]
		if prev.IsZero() {
			out = append(out, decimal.Zero)
			continue
		}
		out = append(out, prices[i].Div(prev).Sub(one))
	}
	return out
}

// CumulativeReturn 期間累積リターン (last/first - 1) を算出する。
func CumulativeReturn(prices []decimal.Decimal) decimal.Decimal {
	if len(prices) < 2 {
		return decimal.Zero
	}
	first := prices[0]
	if first.IsZero() {
		return decimal.Zero
	}
	return prices[len(prices)-1].Div(first).Sub(decimal.NewFromInt(1))
}

// AnnualizedReturn 累積リターンを年率換算する（幾何平均ベース）。
// numReturns は日次リターンの本数（=保有営業日数）。
// (1+cumulativeReturn)^(252/numReturns) - 1
func AnnualizedReturn(cumulativeReturn decimal.Decimal, numReturns int) decimal.Decimal {
	if numReturns <= 0 {
		return decimal.Zero
	}
	base, _ := cumulativeReturn.Add(decimal.NewFromInt(1)).Float64()
	if base <= 0 {
		// 全損（-100%以下）は年率も -100% とする
		return decimal.NewFromInt(-1)
	}
	exp := float64(tradingDaysPerYear) / float64(numReturns)
	return decimal.NewFromFloat(math.Pow(base, exp) - 1)
}

// AnnualizedVolatility 日次リターンの標本標準偏差を年率換算する（×√252）。
func AnnualizedVolatility(dailyReturns []decimal.Decimal) decimal.Decimal {
	if len(dailyReturns) < 2 {
		return decimal.Zero
	}
	return stdDevSample(dailyReturns).Mul(decimalSqrt(decimal.NewFromInt(tradingDaysPerYear)))
}

// AnnualizedDownsideDeviation target を下回る日次リターンのみで下方偏差を求め年率換算する。
// ソルティノレシオの分母に用いる。
func AnnualizedDownsideDeviation(dailyReturns []decimal.Decimal, target decimal.Decimal) decimal.Decimal {
	if len(dailyReturns) < 2 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, r := range dailyReturns {
		if r.LessThan(target) {
			d := r.Sub(target)
			sum = sum.Add(d.Mul(d))
		}
	}
	variance := sum.Div(decimal.NewFromInt(int64(len(dailyReturns) - 1)))
	return decimalSqrt(variance).Mul(decimalSqrt(decimal.NewFromInt(tradingDaysPerYear)))
}

// MaxDrawdown 価格系列のピークからの最大下落率を負の小数で返す（例: -0.23 = 最大23%下落）。
func MaxDrawdown(prices []decimal.Decimal) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}
	peak := prices[0]
	maxDD := decimal.Zero // 最も小さい（最も負の）値を保持
	for _, p := range prices {
		if p.GreaterThan(peak) {
			peak = p
		}
		if peak.IsZero() {
			continue
		}
		dd := p.Sub(peak).Div(peak) // <= 0
		if dd.LessThan(maxDD) {
			maxDD = dd
		}
	}
	return maxDD
}

// SharpeRatio (年率リターン - リスクフリー) / 年率ボラティリティ。
func SharpeRatio(annualizedReturn, annualizedVolatility, riskFreeRate decimal.Decimal) decimal.Decimal {
	if annualizedVolatility.IsZero() {
		return decimal.Zero
	}
	return annualizedReturn.Sub(riskFreeRate).Div(annualizedVolatility)
}

// SortinoRatio (年率リターン - リスクフリー) / 年率下方偏差。
func SortinoRatio(annualizedReturn, annualizedDownsideDeviation, riskFreeRate decimal.Decimal) decimal.Decimal {
	if annualizedDownsideDeviation.IsZero() {
		return decimal.Zero
	}
	return annualizedReturn.Sub(riskFreeRate).Div(annualizedDownsideDeviation)
}

// CalmarRatio 年率リターン / |最大ドローダウン|。
func CalmarRatio(annualizedReturn, maxDrawdown decimal.Decimal) decimal.Decimal {
	absDD := maxDrawdown.Abs()
	if absDD.IsZero() {
		return decimal.Zero
	}
	return annualizedReturn.Div(absDD)
}

// Beta ベンチマークに対するβ（cov(stock, bench) / var(bench)）を算出する。
func Beta(stockReturns, benchmarkReturns []decimal.Decimal) decimal.Decimal {
	n := len(stockReturns)
	if n < 2 || n != len(benchmarkReturns) {
		return decimal.Zero
	}
	benchVar := sampleVariance(benchmarkReturns)
	if benchVar.IsZero() {
		return decimal.Zero
	}
	return sampleCovariance(stockReturns, benchmarkReturns).Div(benchVar)
}

// Correlation 2系列の相関係数（cov / (stdA * stdB)）を算出する。
func Correlation(a, b []decimal.Decimal) decimal.Decimal {
	n := len(a)
	if n < 2 || n != len(b) {
		return decimal.Zero
	}
	sa := stdDevSample(a)
	sb := stdDevSample(b)
	if sa.IsZero() || sb.IsZero() {
		return decimal.Zero
	}
	return sampleCovariance(a, b).Div(sa.Mul(sb))
}

// ExcessReturn 銘柄リターンからベンチマークリターンを差し引いた超過リターン（相対力）。
func ExcessReturn(stockReturn, benchmarkReturn decimal.Decimal) decimal.Decimal {
	return stockReturn.Sub(benchmarkReturn)
}

// --- 内部統計ヘルパー（分散・共分散・標準偏差はいずれも標本ベース: 分母 n-1） ---

func mean(xs []decimal.Decimal) decimal.Decimal {
	if len(xs) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, x := range xs {
		sum = sum.Add(x)
	}
	return sum.Div(decimal.NewFromInt(int64(len(xs))))
}

func sampleVariance(xs []decimal.Decimal) decimal.Decimal {
	n := len(xs)
	if n < 2 {
		return decimal.Zero
	}
	m := mean(xs)
	sum := decimal.Zero
	for _, x := range xs {
		d := x.Sub(m)
		sum = sum.Add(d.Mul(d))
	}
	return sum.Div(decimal.NewFromInt(int64(n - 1)))
}

func stdDevSample(xs []decimal.Decimal) decimal.Decimal {
	return decimalSqrt(sampleVariance(xs))
}

func sampleCovariance(xs, ys []decimal.Decimal) decimal.Decimal {
	n := len(xs)
	if n < 2 || n != len(ys) {
		return decimal.Zero
	}
	mx := mean(xs)
	my := mean(ys)
	sum := decimal.Zero
	for i := range n {
		sum = sum.Add(xs[i].Sub(mx).Mul(ys[i].Sub(my)))
	}
	return sum.Div(decimal.NewFromInt(int64(n - 1)))
}

// decimalSqrt decimal の平方根を float64 経由で算出する（負数・ゼロは 0）。
func decimalSqrt(x decimal.Decimal) decimal.Decimal {
	if x.IsZero() || x.IsNegative() {
		return decimal.Zero
	}
	f, _ := x.Float64()
	return decimal.NewFromFloat(math.Sqrt(f))
}

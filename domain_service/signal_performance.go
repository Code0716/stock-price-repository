package domain_service

import (
	"sort"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
)

// ForwardReturns シグナル日以降の価格系列（昇順）から各 horizon のリターンを計算する。
// prices は呼び出し側（filterPricesFrom）がシグナル日以降（当日含む）に切り取った日足を date 昇順で渡すこと。
// prices[0] が P0（基準価格）となる。baseFound=false は prices が空、または prices[0].Adjclose がゼロのとき。
// その場合 returns は nil。action="Sell" のとき符号を反転する。
func ForwardReturns(prices []*models.StockBrandDailyPrice, action string, horizons []int) (map[int]*decimal.Decimal, bool) {
	if len(prices) == 0 {
		return nil, false
	}

	p0 := prices[0].Adjclose
	if p0.IsZero() {
		return nil, false
	}

	returns := make(map[int]*decimal.Decimal, len(horizons))
	for _, h := range horizons {
		if h < len(prices) {
			r := prices[h].Adjclose.Div(p0).Sub(decimal.NewFromInt(1))
			if action == models.AnalyzeStockBrandPriceHistoryActionSell {
				r = r.Neg()
			}
			r = r.Round(6)
			returns[h] = &r
		} else {
			returns[h] = nil
		}
	}

	return returns, true
}

// AggregateSignalPerformance シグナル明細と各リターンマップから手法別サマリを生成する。
// signalReturns[i] は signals[i] に対応し、key=horizon, value=nil は未到来を意味する。
func AggregateSignalPerformance(signals []*models.EvaluatedSignal, horizons []int) []*models.SignalPerformanceSummary {
	type methodBucket struct {
		signalCount  int
		skippedCount int
		returns      map[int][]decimal.Decimal
	}

	buckets := make(map[string]*methodBucket)

	for _, sig := range signals {
		method := sig.Method
		b, ok := buckets[method]
		if !ok {
			b = &methodBucket{returns: make(map[int][]decimal.Decimal)}
			buckets[method] = b
		}

		if sig.Returns == nil {
			b.skippedCount++
			b.signalCount++
			continue
		}
		b.signalCount++
		for _, h := range horizons {
			if v := sig.Returns[h]; v != nil {
				b.returns[h] = append(b.returns[h], *v)
			}
		}
	}

	summaries := make([]*models.SignalPerformanceSummary, 0, len(buckets))
	for method, b := range buckets {
		stats := make(map[int]*models.HorizonStats, len(horizons))
		for _, h := range horizons {
			rs := b.returns[h]
			stats[h] = calcHorizonStats(rs)
		}
		summaries = append(summaries, &models.SignalPerformanceSummary{
			Method:       method,
			SignalCount:  b.signalCount,
			SkippedCount: b.skippedCount,
			Stats:        stats,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Method < summaries[j].Method
	})

	return summaries
}

func calcHorizonStats(returns []decimal.Decimal) *models.HorizonStats {
	n := len(returns)
	if n == 0 {
		return &models.HorizonStats{}
	}

	winCount := 0
	sum := decimal.Zero
	best := returns[0]
	worst := returns[0]
	sorted := make([]decimal.Decimal, n)
	copy(sorted, returns)

	for _, r := range returns {
		if r.IsPositive() {
			winCount++
		}
		sum = sum.Add(r)
		if r.GreaterThan(best) {
			best = r
		}
		if r.LessThan(worst) {
			worst = r
		}
	}

	avg := sum.Div(decimal.NewFromInt(int64(n)))

	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LessThan(sorted[j]) })
	var median decimal.Decimal
	if n%2 == 0 {
		median = sorted[n/2-1].Add(sorted[n/2]).Div(decimal.NewFromInt(2))
	} else {
		median = sorted[n/2]
	}

	winRate := decimal.NewFromInt(int64(winCount)).Div(decimal.NewFromInt(int64(n))).Round(4)

	return &models.HorizonStats{
		EvaluatedCount: n,
		WinCount:       winCount,
		WinRate:        winRate,
		AvgReturn:      avg.Round(6),
		MedianReturn:   median.Round(6),
		BestReturn:     best.Round(6),
		WorstReturn:    worst.Round(6),
	}
}

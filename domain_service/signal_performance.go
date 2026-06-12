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

// rankBandDefs はランク帯の定義（帯名、min/max でシグナルを仕分け）
type rankBandDef struct {
	band string
	min  int
	max  int // -1 = 上限なし
}

var rankBandDefs = []rankBandDef{
	{"1-3", 1, 3},
	{"4-10", 4, 10},
	{"11+", 11, -1},
}

// AggregateByRankBand SignalRank 非nilのシグナルをランク帯別に集計する。
// 帯の順序は固定（1-3 / 4-10 / 11+）。対象0件の帯も SignalCount=0 で返す。
func AggregateByRankBand(signals []*models.EvaluatedSignal, horizons []int) []*models.BandSummary {
	type bucket struct {
		signals []*models.EvaluatedSignal
	}
	buckets := make([]bucket, len(rankBandDefs))

	for _, sig := range signals {
		if sig.SignalRank == nil {
			continue
		}
		rank := *sig.SignalRank
		for i, def := range rankBandDefs {
			if rank >= def.min && (def.max == -1 || rank <= def.max) {
				buckets[i].signals = append(buckets[i].signals, sig)
				break
			}
		}
	}

	result := make([]*models.BandSummary, len(rankBandDefs))
	for i, def := range rankBandDefs {
		sigs := buckets[i].signals
		stats := calcBandStats(sigs, horizons)
		result[i] = &models.BandSummary{
			Band:        def.band,
			SignalCount: len(sigs),
			Stats:       stats,
		}
	}
	return result
}

// AggregateByScoreQuartile Score 非nilのシグナルをスコア四分位別に集計する。
// score 昇順で四分位に分割（Q1=下位〜Q4=上位、境界は件数ベースの等分割）。
// n<4 のときは空スライスを返す。
func AggregateByScoreQuartile(signals []*models.EvaluatedSignal, horizons []int) []*models.BandSummary {
	// Score 非nilのみ抽出
	scored := make([]*models.EvaluatedSignal, 0, len(signals))
	for _, sig := range signals {
		if sig.Score != nil {
			scored = append(scored, sig)
		}
	}

	if len(scored) < 4 {
		return []*models.BandSummary{}
	}

	// score 昇順ソート
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score.LessThan(*scored[j].Score)
	})

	n := len(scored)
	result := make([]*models.BandSummary, 4)
	for q := 0; q < 4; q++ {
		// 件数ベースの等分割: [q*n/4, (q+1)*n/4)
		lo := q * n / 4
		hi := (q + 1) * n / 4
		if q == 3 {
			hi = n // 最後の四分位は端数を全て含む
		}
		qSigs := scored[lo:hi]
		stats := calcBandStats(qSigs, horizons)
		result[q] = &models.BandSummary{
			Band:        "Q" + string(rune('1'+q)),
			SignalCount: len(qSigs),
			Stats:       stats,
		}
	}
	return result
}

// calcBandStats シグナル群から horizon 別統計を生成する（Returns nil = skip）
func calcBandStats(signals []*models.EvaluatedSignal, horizons []int) map[int]*models.HorizonStats {
	returns := make(map[int][]decimal.Decimal, len(horizons))
	for _, sig := range signals {
		if sig.Returns == nil {
			continue
		}
		for _, h := range horizons {
			if v := sig.Returns[h]; v != nil {
				returns[h] = append(returns[h], *v)
			}
		}
	}

	stats := make(map[int]*models.HorizonStats, len(horizons))
	for _, h := range horizons {
		stats[h] = calcHorizonStats(returns[h])
	}
	return stats
}

package domain_service

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// DailyPickFilterParams 事前フィルタの閾値。
type DailyPickFilterParams struct {
	WindowDays         int             // 戦略・指標のウォームアップに必要な営業日数（120）
	MetricsWindowDays  int             // 出来高倍率・平均売買代金を計算する営業日数（20）
	MinClosePrice      decimal.Decimal // 株価下限（300円）
	MinAvgTradingValue decimal.Decimal // 直近平均売買代金の下限（1億円）
	MinAvgVolume       decimal.Decimal // 直近平均出来高の下限（50,000株）
}

// DefaultDailyPickFilterParams 標準の事前フィルタ閾値。
func DefaultDailyPickFilterParams() DailyPickFilterParams {
	return DailyPickFilterParams{
		WindowDays:         120,
		MetricsWindowDays:  20,
		MinClosePrice:      decimal.RequireFromString("300"),
		MinAvgTradingValue: decimal.RequireFromString("100000000"),
		MinAvgVolume:       decimal.RequireFromString("50000"),
	}
}

// DailyPickMetrics 1銘柄の最新営業日時点の因子生値（daily_stock_pick のスコア入力カラムと1:1対応）。
type DailyPickMetrics struct {
	SignalCount     int
	VolumeRatio     decimal.Decimal
	AvgTradingValue decimal.Decimal
	ADX             decimal.Decimal
	PlusDI          decimal.Decimal
	MinusDI         decimal.Decimal
	ATRRatio        decimal.Decimal
	RSI             decimal.Decimal
	Close           decimal.Decimal
	AdjClose        decimal.Decimal
}

// DailyPickCandidate スクリーニング通過銘柄（スコア済み）。
type DailyPickCandidate struct {
	Brand      *models.StockBrand
	Metrics    DailyPickMetrics
	Score      decimal.Decimal
	Strategies []string // 点灯した基本戦略キー（StrategyOrder順）
}

// EvaluateDailyPickCandidate 1銘柄の日足（date昇順、末尾が最新営業日のバー）から候補を評価する。
// 事前フィルタに落ちた、または基本4戦略が1つも点灯していない場合は nil を返す。
// prices の長さは filter.WindowDays 以上であることを呼び出し側が保証すること。
func EvaluateDailyPickCandidate(
	brand *models.StockBrand,
	prices []*models.StockBrandDailyPrice,
	filter DailyPickFilterParams,
	weights DailyPickScoreWeights,
) *DailyPickCandidate {
	n := len(prices)
	if n < filter.WindowDays {
		return nil
	}
	last := prices[n-1]
	if !passesDailyPickHardFilters(prices, filter) {
		return nil
	}

	adx, ok := lastADX(prices)
	if !ok || adx.PlusDI.LessThanOrEqual(adx.MinusDI) {
		return nil // ADXが未確定、または下降トレンド中の点灯は買い候補にしない
	}

	rsi, atrRatio, ok := lastRSIAndATRRatio(prices, last.Close)
	if !ok {
		return nil
	}

	strategies := detectDailyPickStrategies(prices)
	if len(strategies) == 0 {
		return nil
	}

	metricsStart := n - filter.MetricsWindowDays
	if metricsStart < 0 {
		metricsStart = 0
	}
	avgTradingValue := windowAvgTradingValue(prices[metricsStart:n])

	// 出来高倍率は「当日出来高 ÷ 直前(当日を含まない)平均出来高」。既存の avgVolume ヘルパーを再利用する。
	volumeRatio := decimal.Zero
	if baseline := avgVolume(prices, n-1, filter.MetricsWindowDays); !baseline.IsZero() {
		volumeRatio = decimal.NewFromInt(last.Volume).Div(baseline)
	}

	metrics := DailyPickMetrics{
		SignalCount:     len(strategies),
		VolumeRatio:     volumeRatio,
		AvgTradingValue: avgTradingValue,
		ADX:             adx.ADX,
		PlusDI:          adx.PlusDI,
		MinusDI:         adx.MinusDI,
		ATRRatio:        atrRatio,
		RSI:             rsi,
		Close:           last.Close,
		AdjClose:        last.Adjclose,
	}

	return &DailyPickCandidate{
		Brand:      brand,
		Metrics:    metrics,
		Score:      ScoreDailyPick(metrics, weights),
		Strategies: strategies,
	}
}

// passesDailyPickHardFilters 株価・値幅・流動性の事前フィルタを検証する。
func passesDailyPickHardFilters(prices []*models.StockBrandDailyPrice, filter DailyPickFilterParams) bool {
	n := len(prices)
	last := prices[n-1]
	if last.Close.LessThan(filter.MinClosePrice) {
		return false
	}
	if last.High.Equal(last.Low) {
		return false // ストップ高等の値幅ゼロは翌日買えないため除外
	}

	metricsStart := n - filter.MetricsWindowDays
	if metricsStart < 0 {
		metricsStart = 0
	}
	liquidityWindow := prices[metricsStart:n]
	if windowAvgTradingValue(liquidityWindow).LessThan(filter.MinAvgTradingValue) {
		return false
	}
	if windowAvgVolume(liquidityWindow).LessThan(filter.MinAvgVolume) {
		return false
	}
	return true
}

// lastADX 最新営業日時点のADXを返す。未確定なら ok=false。
func lastADX(prices []*models.StockBrandDailyPrice) (ADXResult, bool) {
	adxResults := CalculateADX(prices, 14)
	if adxResults == nil {
		return ADXResult{}, false
	}
	return adxResults[len(prices)-1], true
}

// lastRSIAndATRRatio 最新営業日時点のRSIとATR比率（ATR/終値）を返す。未確定なら ok=false。
func lastRSIAndATRRatio(prices []*models.StockBrandDailyPrice, lastClose decimal.Decimal) (rsi, atrRatio decimal.Decimal, ok bool) {
	closes := ExtractClosePrices(prices)
	rsiSeries := CalculateRSI(closes, 14)
	atrSeries := CalculateATR(prices, 14)
	if rsiSeries == nil || atrSeries == nil {
		return decimal.Zero, decimal.Zero, false
	}
	n := len(prices)
	atrRatio = decimal.Zero
	if !lastClose.IsZero() {
		atrRatio = atrSeries[n-1].Div(lastClose)
	}
	return rsiSeries[n-1], atrRatio, true
}

// detectDailyPickStrategies 最新営業日に点灯した基本4戦略のキーを返す（StrategyOrder順）。
func detectDailyPickStrategies(prices []*models.StockBrandDailyPrice) []string {
	n := len(prices)
	var strategies []string
	for _, s := range DailyPickBaseStrategies {
		signals := EntrySignalsByStrategy(s, prices)
		if len(signals) == n && signals[n-1] {
			strategies = append(strategies, s)
		}
	}
	return strategies
}

// RankDailyPickCandidates スコア降順（同点は TickerSymbol 昇順）に並べ、
// 同一 Sector33CodeName の採用数を maxPerSector 以下に抑えつつ上位 topN を選ぶ。
// maxPerSector <= 0 で無制限。Sector33CodeName が空の銘柄は上限の対象外。
func RankDailyPickCandidates(candidates []*DailyPickCandidate, pickDate time.Time, topN, maxPerSector int) []*models.DailyStockPick {
	sorted := make([]*DailyPickCandidate, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].Score.Equal(sorted[j].Score) {
			return sorted[i].Score.GreaterThan(sorted[j].Score)
		}
		return sorted[i].Brand.TickerSymbol < sorted[j].Brand.TickerSymbol
	})

	sectorCount := make(map[string]int)
	picks := make([]*models.DailyStockPick, 0, topN)
	for _, c := range sorted {
		if len(picks) >= topN {
			break
		}
		sector := c.Brand.Sector33CodeName
		if maxPerSector > 0 && sector != "" && sectorCount[sector] >= maxPerSector {
			continue
		}
		picks = append(picks, candidateToDailyStockPick(c, pickDate, len(picks)+1))
		if sector != "" {
			sectorCount[sector]++
		}
	}
	return picks
}

func candidateToDailyStockPick(c *DailyPickCandidate, pickDate time.Time, rank int) *models.DailyStockPick {
	return &models.DailyStockPick{
		PickDate:          pickDate,
		StockBrandID:      c.Brand.ID,
		TickerSymbol:      c.Brand.TickerSymbol,
		Name:              c.Brand.Name,
		PickRank:          rank,
		Score:             c.Score,
		ScoreVersion:      DailyPickScoreVersion,
		SignalCount:       c.Metrics.SignalCount,
		Strategies:        c.Strategies,
		Sector33CodeName:  c.Brand.Sector33CodeName,
		BaseClosePrice:    c.Metrics.Close,
		BaseAdjClosePrice: c.Metrics.AdjClose,
		AvgTradingValue:   c.Metrics.AvgTradingValue,
		VolumeRatio:       c.Metrics.VolumeRatio,
		ADX:               c.Metrics.ADX,
		PlusDI:            c.Metrics.PlusDI,
		MinusDI:           c.Metrics.MinusDI,
		ATRRatio:          c.Metrics.ATRRatio,
		RSI:               c.Metrics.RSI,
	}
}

// windowAvgTradingValue 区間（当日含む）の平均売買代金 (volume*close)。
func windowAvgTradingValue(prices []*models.StockBrandDailyPrice) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, p := range prices {
		sum = sum.Add(decimal.NewFromInt(p.Volume).Mul(p.Close))
	}
	return sum.Div(decimal.NewFromInt(int64(len(prices))))
}

// windowAvgVolume 区間（当日含む）の平均出来高。
func windowAvgVolume(prices []*models.StockBrandDailyPrice) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, p := range prices {
		sum = sum.Add(decimal.NewFromInt(p.Volume))
	}
	return sum.Div(decimal.NewFromInt(int64(len(prices))))
}

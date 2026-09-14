package domain_service

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

const (
	// DailyAvoidRuleVersion 判定ルール定義バージョン。閾値変更時にインクリメントし過去分と混ぜて集計しない。
	DailyAvoidRuleVersion = "v1"
	// DailyAvoidVolatilityWindowDays 実現ボラティリティの算出に使う営業日数（直近12ヶ月）。
	DailyAvoidVolatilityWindowDays = 252
	// DailyAvoidMinReturns これ未満の日次リターン本数しかない銘柄（新規上場等）は判定不能として除外する。
	DailyAvoidMinReturns = 200
)

// DailyAvoidFlagParams 避けるべき銘柄の閾値パラメータ。
type DailyAvoidFlagParams struct {
	FlagPercentile decimal.Decimal // 0.20 = ボラ上位20%までをフラグ対象にする
	HighPercentile decimal.Decimal // 0.10 = ボラ上位10%以内は severity=high
}

// DefaultDailyAvoidFlagParams 実データ検証で確認した既定閾値（上位20%をフラグ、うち上位10%はhigh）。
func DefaultDailyAvoidFlagParams() DailyAvoidFlagParams {
	return DailyAvoidFlagParams{
		FlagPercentile: decimal.RequireFromString("0.20"),
		HighPercentile: decimal.RequireFromString("0.10"),
	}
}

// AvoidCandidate 流動性ユニバースを通過した1銘柄の、順位付け前の生値。
type AvoidCandidate struct {
	Brand           *models.StockBrand
	Volatility12M   decimal.Decimal // 直近252営業日の日次リターン標準偏差を年率換算した値
	AvgTradingValue decimal.Decimal
	BaseClosePrice  decimal.Decimal
}

// EvaluateAvoidCandidate 1銘柄の日足（date昇順、末尾が as_of_date のバー）からボラ候補を評価する。
// 流動性ユニバースに入らない、または直近12ヶ月分の日次リターンが DailyAvoidMinReturns 未満（新規上場等で
// 判定不能）の場合は nil を返す。この時点ではまだユニバース内の順位は決まらない。
//
// 必ず Adjclose（分割・併合調整後終値）でリターンを計算すること。Close を使うと分割銘柄が
// 偽の最高ボラに化け、実データ検証の結論と乖離する。
func EvaluateAvoidCandidate(brand *models.StockBrand, prices []*models.StockBrandDailyPrice, filter DailyPickFilterParams) *AvoidCandidate {
	n := len(prices)
	if n == 0 {
		return nil
	}
	if !PassesLiquidityUniverse(prices, filter) {
		return nil
	}

	// DailyReturns は隣接差分なので、252本のリターンを得るには253本の価格が要る。
	windowStart := n - DailyAvoidVolatilityWindowDays - 1
	if windowStart < 0 {
		windowStart = 0
	}
	returns := DailyReturns(ExtractAdjClosePrices(prices[windowStart:n]))
	if len(returns) < DailyAvoidMinReturns {
		return nil
	}

	metricsStart := n - filter.MetricsWindowDays
	if metricsStart < 0 {
		metricsStart = 0
	}

	last := prices[n-1]
	return &AvoidCandidate{
		Brand:           brand,
		Volatility12M:   AnnualizedVolatility(returns),
		AvgTradingValue: windowAvgTradingValue(prices[metricsStart:n]),
		BaseClosePrice:  last.Close,
	}
}

// RankAvoidCandidates ボラ降順（同値は TickerSymbol 昇順）で順位付けし、
// 上位 FlagPercentile までを models.DailyAvoidStock に変換して返す。
// universeSize / thresholdVolatility は全行に同じ値を詰める。候補が空なら空スライスを返す。
func RankAvoidCandidates(candidates []*AvoidCandidate, asOfDate time.Time, params DailyAvoidFlagParams) []*models.DailyAvoidStock {
	universeSize := len(candidates)
	if universeSize == 0 {
		return nil
	}

	sorted := make([]*AvoidCandidate, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].Volatility12M.Equal(sorted[j].Volatility12M) {
			return sorted[i].Volatility12M.GreaterThan(sorted[j].Volatility12M)
		}
		return sorted[i].Brand.TickerSymbol < sorted[j].Brand.TickerSymbol
	})

	flagCount := ceilPercentile(universeSize, params.FlagPercentile)
	if flagCount == 0 {
		return nil
	}
	highCount := ceilPercentile(universeSize, params.HighPercentile)
	thresholdVolatility := sorted[flagCount-1].Volatility12M

	out := make([]*models.DailyAvoidStock, 0, flagCount)
	for i := range flagCount {
		c := sorted[i]
		severity := models.DailyAvoidSeverityElevated
		if i < highCount {
			severity = models.DailyAvoidSeverityHigh
		}
		out = append(out, &models.DailyAvoidStock{
			AsOfDate:             asOfDate,
			StockBrandID:         c.Brand.ID,
			TickerSymbol:         c.Brand.TickerSymbol,
			Name:                 c.Brand.Name,
			AvoidRank:            i + 1,
			Severity:             severity,
			Reason:               models.DailyAvoidReasonHighVolatility,
			RuleVersion:          DailyAvoidRuleVersion,
			Volatility12M:        c.Volatility12M,
			VolatilityPercentile: percentileOf(i, universeSize),
			UniverseSize:         universeSize,
			ThresholdVolatility:  thresholdVolatility,
			AvgTradingValue:      c.AvgTradingValue,
			BaseClosePrice:       c.BaseClosePrice,
			Sector33CodeName:     c.Brand.Sector33CodeName,
		})
	}
	return out
}

// ceilPercentile universeSize 件のうち pct（0.20等）に相当する件数を切り上げで返す。
func ceilPercentile(universeSize int, pct decimal.Decimal) int {
	if universeSize <= 0 {
		return 0
	}
	n := decimal.NewFromInt(int64(universeSize)).Mul(pct).Ceil().IntPart()
	if int(n) > universeSize {
		return universeSize
	}
	return int(n)
}

// percentileOf ボラ降順で0始まりの順位 rankIdx（0=最高ボラ）が、ユニバース内の上位何割にあたるかを返す。
func percentileOf(rankIdx, universeSize int) decimal.Decimal {
	return decimal.NewFromInt(int64(rankIdx + 1)).Div(decimal.NewFromInt(int64(universeSize))).Round(4)
}

// SummarizeDailyAvoidStocks 1日分の避けるべき銘柄からサマリを作る。rows は avoid_rank 昇順を想定。
// 空なら全フィールドゼロ値のサマリを返す（nil にはしない。API レスポンスで null にしないため）。
func SummarizeDailyAvoidStocks(rows []*models.DailyAvoidStock) models.DailyAvoidStockSummary {
	s := models.DailyAvoidStockSummary{
		MaxVolatility:        decimal.Zero,
		MinFlaggedVolatility: decimal.Zero,
	}
	if len(rows) == 0 {
		return s
	}

	s.FlaggedCount = len(rows)
	s.MaxVolatility = rows[0].Volatility12M
	s.MinFlaggedVolatility = rows[0].Volatility12M
	for _, r := range rows {
		if r.Severity == models.DailyAvoidSeverityHigh {
			s.HighCount++
		} else {
			s.ElevatedCount++
		}
		if r.Volatility12M.GreaterThan(s.MaxVolatility) {
			s.MaxVolatility = r.Volatility12M
		}
		if r.Volatility12M.LessThan(s.MinFlaggedVolatility) {
			s.MinFlaggedVolatility = r.Volatility12M
		}
	}
	return s
}

package domain_service

import (
	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
)

// 各戦略のエントリー（買い）シグナルを日次の []bool で返す純粋関数群。
// 条件式は stt-golang の各 find_*_stock usecase から移植している。
// 返り値の長さは len(prices) と一致し、true はその日の終値時点でエントリー条件成立を表す。

// 戦略識別子（フロント表示・ランキングのキーに使う）
const (
	StrategyMACDBullish        = "macd_bullish"
	StrategyBollingerBreakout  = "bollinger_breakout"
	StrategyTriangleFormation  = "triangle_formation"
	StrategyMovingAverageCross = "ma_cross"
	StrategyMultipleSignals    = "multiple_signals"
)

// StrategyLabels 戦略の日本語表示名
var StrategyLabels = map[string]string{
	StrategyMACDBullish:        "MACD強気",
	StrategyBollingerBreakout:  "ボリンジャーブレイク",
	StrategyTriangleFormation:  "三角持ち合い",
	StrategyMovingAverageCross: "移動平均(5/25/75)上抜け",
	StrategyMultipleSignals:    "複数シグナル(2つ以上)",
}

// StrategyOrder ランキング表示・全戦略走査の順序
var StrategyOrder = []string{
	StrategyMACDBullish,
	StrategyBollingerBreakout,
	StrategyTriangleFormation,
	StrategyMovingAverageCross,
	StrategyMultipleSignals,
}

// EntrySignalsByStrategy 指定戦略のエントリーシグナルを返す。
func EntrySignalsByStrategy(strategy string, prices []*models.StockBrandDailyPrice) []bool {
	switch strategy {
	case StrategyMACDBullish:
		return MACDBullishEntrySignals(prices)
	case StrategyBollingerBreakout:
		return BollingerBreakoutEntrySignals(prices)
	case StrategyTriangleFormation:
		return TriangleFormationEntrySignals(prices)
	case StrategyMovingAverageCross:
		return MovingAverageCrossEntrySignals(prices)
	case StrategyMultipleSignals:
		return MultipleSignalsEntrySignals(prices)
	default:
		return make([]bool, len(prices))
	}
}

// MACDBullishEntrySignals MACDゴールデンクロス(Signal≥0) + RSI<70 + 出来高>5日平均。
func MACDBullishEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	macd := CalculateMACD(closes, 12, 26, 9)
	rsi := CalculateRSI(closes, 14)
	if macd == nil || rsi == nil {
		return signals
	}
	// MACD/Signal が有効になる十分なウォームアップ後から評価
	const minIdx = 35
	for i := minIdx; i < n; i++ {
		cur := macd[i]
		prev := macd[i-1]
		goldenCross := cur.MACD.GreaterThan(cur.Signal) &&
			!prev.MACD.GreaterThan(prev.Signal) &&
			cur.Signal.GreaterThanOrEqual(decimal.Zero)
		if !goldenCross {
			continue
		}
		if rsi[i].GreaterThanOrEqual(decimal.NewFromInt(70)) {
			continue
		}
		if decimal.NewFromInt(prices[i].Volume).LessThanOrEqual(avgVolume(prices, i, 5)) {
			continue
		}
		signals[i] = true
	}
	return signals
}

// BollingerBreakoutEntrySignals アッパーバンド上抜け + スクイーズ + 出来高急増 + RSI<70。
func BollingerBreakoutEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	const bbPeriod = 20
	bb := CalculateBollingerBands(closes, bbPeriod, decimal.NewFromInt(2))
	rsi := CalculateRSI(closes, 14)
	if bb == nil || rsi == nil {
		return signals
	}

	// バンド幅の短期(5)/長期(20)平均を比較するため最低 bbPeriod*2 のデータが要る
	minIdx := bbPeriod*2 - 1
	for i := minIdx; i < n; i++ {
		breakout := closes[i].GreaterThan(bb[i].Upper) &&
			!closes[i-1].GreaterThan(bb[i-1].Upper)
		if !breakout {
			continue
		}
		shortAvgBW := avgBandWidth(bb, i, 5)
		longAvgBW := avgBandWidth(bb, i, 20)
		if !shortAvgBW.LessThan(longAvgBW) { // スクイーズ収束からの拡大
			continue
		}
		if decimal.NewFromInt(prices[i].Volume).LessThanOrEqual(avgVolume(prices, i, 5).Mul(decimal.NewFromFloat(1.5))) {
			continue
		}
		if rsi[i].GreaterThanOrEqual(decimal.NewFromInt(70)) {
			continue
		}
		signals[i] = true
	}
	return signals
}

// TriangleFormationEntrySignals 直近窓で高値が下降・安値が上昇（三角持ち合い）+ 出来高>10万。
func TriangleFormationEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	const window = 60
	n := len(prices)
	signals := make([]bool, n)

	for i := window - 1; i < n; i++ {
		start := i - window + 1
		highSlope, okH := localExtremaSlope(prices[start:i+1], true)
		lowSlope, okL := localExtremaSlope(prices[start:i+1], false)
		if !okH || !okL {
			continue
		}
		if highSlope.IsNegative() && lowSlope.IsPositive() && prices[i].Volume > 100000 {
			signals[i] = true
		}
	}
	return signals
}

// MovingAverageCrossEntrySignals 終値が5/25/75日線を全て上抜けた瞬間（前日は全上抜けでない）。
func MovingAverageCrossEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	sma5 := smaSeries(closes, 5)
	sma25 := smaSeries(closes, 25)
	sma75 := smaSeries(closes, 75)

	aboveAll := func(idx int) bool {
		if sma5[idx].IsZero() || sma25[idx].IsZero() || sma75[idx].IsZero() {
			return false
		}
		return closes[idx].GreaterThan(sma5[idx]) &&
			closes[idx].GreaterThan(sma25[idx]) &&
			closes[idx].GreaterThan(sma75[idx])
	}

	for i := 75; i < n; i++ {
		if aboveAll(i) && !aboveAll(i-1) {
			signals[i] = true
		}
	}
	return signals
}

// MultipleSignalsEntrySignals 個別戦略のうち同日に2つ以上成立した日。
func MultipleSignalsEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	macd := MACDBullishEntrySignals(prices)
	bb := BollingerBreakoutEntrySignals(prices)
	tri := TriangleFormationEntrySignals(prices)
	ma := MovingAverageCrossEntrySignals(prices)

	signals := make([]bool, n)
	for i := 0; i < n; i++ {
		count := 0
		if macd[i] {
			count++
		}
		if bb[i] {
			count++
		}
		if tri[i] {
			count++
		}
		if ma[i] {
			count++
		}
		if count >= 2 {
			signals[i] = true
		}
	}
	return signals
}

// --- ヘルパー ---

// avgVolume index i の直前 window 日（i は含まない）の平均出来高。
func avgVolume(prices []*models.StockBrandDailyPrice, i, window int) decimal.Decimal {
	start := i - window
	if start < 0 {
		start = 0
	}
	if start >= i {
		return decimal.Zero
	}
	sum := decimal.Zero
	cnt := 0
	for j := start; j < i; j++ {
		sum = sum.Add(decimal.NewFromInt(prices[j].Volume))
		cnt++
	}
	if cnt == 0 {
		return decimal.Zero
	}
	return sum.Div(decimal.NewFromInt(int64(cnt)))
}

// avgBandWidth index i を含む直近 window 日のバンド幅平均。
func avgBandWidth(bb []BollingerBandsResult, i, window int) decimal.Decimal {
	start := i - window + 1
	if start < 0 {
		start = 0
	}
	sum := decimal.Zero
	cnt := 0
	for j := start; j <= i; j++ {
		sum = sum.Add(bb[j].BandWidth)
		cnt++
	}
	if cnt == 0 {
		return decimal.Zero
	}
	return sum.Div(decimal.NewFromInt(int64(cnt)))
}

// smaSeries 各 index の単純移動平均。period 未満の index は decimal.Zero。
func smaSeries(prices []decimal.Decimal, period int) []decimal.Decimal {
	out := make([]decimal.Decimal, len(prices))
	if len(prices) < period {
		return out
	}
	for i := period - 1; i < len(prices); i++ {
		sum := decimal.Zero
		for j := i - period + 1; j <= i; j++ {
			sum = sum.Add(prices[j])
		}
		out[i] = sum.Div(decimal.NewFromInt(int64(period)))
	}
	return out
}

// localExtremaSlope 窓内の局所極値（high=true なら High の極大、false なら Low の極小）に
// 最小二乗回帰を当てて傾きを返す。極値が2点未満なら ok=false。
func localExtremaSlope(window []*models.StockBrandDailyPrice, high bool) (decimal.Decimal, bool) {
	type point struct {
		x int
		y decimal.Decimal
	}
	var pts []point
	for j := 1; j < len(window)-1; j++ {
		var v, prev, next decimal.Decimal
		if high {
			v, prev, next = window[j].High, window[j-1].High, window[j+1].High
			if v.GreaterThanOrEqual(prev) && v.GreaterThanOrEqual(next) {
				pts = append(pts, point{j, v})
			}
		} else {
			v, prev, next = window[j].Low, window[j-1].Low, window[j+1].Low
			if v.LessThanOrEqual(prev) && v.LessThanOrEqual(next) {
				pts = append(pts, point{j, v})
			}
		}
	}
	if len(pts) < 2 {
		return decimal.Zero, false
	}

	nn := decimal.NewFromInt(int64(len(pts)))
	sumX, sumY, sumXY, sumXX := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	for _, p := range pts {
		x := decimal.NewFromInt(int64(p.x))
		sumX = sumX.Add(x)
		sumY = sumY.Add(p.y)
		sumXY = sumXY.Add(x.Mul(p.y))
		sumXX = sumXX.Add(x.Mul(x))
	}
	denom := nn.Mul(sumXX).Sub(sumX.Mul(sumX))
	if denom.IsZero() {
		return decimal.Zero, false
	}
	// slope = (n*Σxy - Σx*Σy) / (n*Σx² - (Σx)²)
	slope := nn.Mul(sumXY).Sub(sumX.Mul(sumY)).Div(denom)
	return slope, true
}

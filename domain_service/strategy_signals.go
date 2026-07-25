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
	StrategyTriangleFormation:  "三角持ち合いブレイク",
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

// ExitSignalsByStrategy 指定戦略の反転（手仕舞い）シグナルを日次の []bool で返す純粋関数。
// 返り値の長さは len(prices) と一致し、true はその日の終値時点でイグジット条件成立を表す。
// 未知の strategy は全 false を返す。
func ExitSignalsByStrategy(strategy string, prices []*models.StockBrandDailyPrice) []bool {
	switch strategy {
	case StrategyMACDBullish:
		return MACDBullishExitSignals(prices)
	case StrategyBollingerBreakout:
		return BollingerBreakoutExitSignals(prices)
	case StrategyTriangleFormation:
		return GenericTrendBreakExitSignals(prices)
	case StrategyMovingAverageCross:
		return MovingAverageCrossExitSignals(prices)
	case StrategyMultipleSignals:
		return GenericTrendBreakExitSignals(prices)
	default:
		return make([]bool, len(prices))
	}
}

// MACDBullishExitSignals MACDデッドクロス（前日 MACD >= Signal かつ当日 MACD < Signal）で手仕舞い。
// MACDBullishEntrySignals（ゴールデンクロス）の反転シグナル。
func MACDBullishExitSignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	macd := CalculateMACD(closes, 12, 26, 9)
	if macd == nil {
		return signals
	}
	// MACD/Signal が有効になる十分なウォームアップ後から評価（エントリーと同じ閾値）
	const minIdx = 35
	for i := minIdx; i < n; i++ {
		cur := macd[i]
		prev := macd[i-1]
		// デッドクロス: 前日 MACD >= Signal かつ当日 MACD < Signal
		deadCross := cur.MACD.LessThan(cur.Signal) &&
			!prev.MACD.LessThan(prev.Signal)
		if deadCross {
			signals[i] = true
		}
	}
	return signals
}

// BollingerBreakoutExitSignals 終値がボリンジャーミドルバンド（20SMA）を下抜け。
// 前日 close >= middle かつ当日 close < middle の成立を手仕舞いシグナルとする。
// BollingerBreakoutEntrySignals（アッパー上抜け）の反転シグナル。
func BollingerBreakoutExitSignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	const bbPeriod = 20
	bb := CalculateBollingerBands(closes, bbPeriod, decimal.NewFromInt(2))
	if bb == nil {
		return signals
	}

	// ミドルバンドが有効になる bbPeriod-1 日目以降から評価
	for i := bbPeriod; i < n; i++ {
		// 終値がミドルバンド（20SMA）を下抜け
		belowMiddle := closes[i].LessThan(bb[i].Middle) &&
			!closes[i-1].LessThan(bb[i-1].Middle)
		if belowMiddle {
			signals[i] = true
		}
	}
	return signals
}

// MovingAverageCrossExitSignals 5日SMAが25日SMAを下抜け（前日 5SMA >= 25SMA かつ当日 5SMA < 25SMA）。
// MovingAverageCrossEntrySignals（5/25/75全上抜け）の反転シグナル。
func MovingAverageCrossExitSignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	sma5 := smaSeries(closes, 5)
	sma25 := smaSeries(closes, 25)

	// sma25 が有効になる index=24 以降、かつ前日も有効な index=25 以降から評価
	for i := 25; i < n; i++ {
		if sma5[i].IsZero() || sma25[i].IsZero() || sma5[i-1].IsZero() || sma25[i-1].IsZero() {
			continue
		}
		// 5SMA が 25SMA を下抜け
		crossUnder := sma5[i].LessThan(sma25[i]) &&
			!sma5[i-1].LessThan(sma25[i-1])
		if crossUnder {
			signals[i] = true
		}
	}
	return signals
}

// GenericTrendBreakExitSignals 汎用トレンド破れルール: 終値が25日SMAを下抜け。
// TriangleFormation・MultipleSignals など固有の反転シグナルが定義しにくい戦略に使用する。
func GenericTrendBreakExitSignals(prices []*models.StockBrandDailyPrice) []bool {
	n := len(prices)
	signals := make([]bool, n)
	closes := ExtractClosePrices(prices)

	sma25 := smaSeries(closes, 25)

	// sma25 が有効になる index=24 以降、かつ前日も有効な index=25 以降から評価
	for i := 25; i < n; i++ {
		if sma25[i].IsZero() || sma25[i-1].IsZero() {
			continue
		}
		// 終値が 25SMA を下抜け
		belowSMA25 := closes[i].LessThan(sma25[i]) &&
			!closes[i-1].LessThan(sma25[i-1])
		if belowSMA25 {
			signals[i] = true
		}
	}
	return signals
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

// TriangleFormationParams 三角持ち合いブレイクの判定パラメータ。全銘柄共通でパッケージ変数に集約し、
// チューニングを一箇所に閉じる。
type TriangleFormationParams struct {
	Window                int             // 収縮を測る窓（当日を含まない）
	ExtremaStrength       int             // 局所極値の左右本数（fractal強度）
	MinExtremaPoints      int             // 傾き回帰に必要な極値点数
	MinSlopeRatio         decimal.Decimal // 傾きの最小絶対値（当日終値比・1日あたり）
	RangeContractionRatio decimal.Decimal // 窓後半1/3の平均日中レンジ ÷ 前半1/3 の上限
	BreakLookback         int             // ブレイク判定に使う直近高値の日数
	VolumeWindow          int             // 出来高平均の窓
	VolumeMultiplier      decimal.Decimal // 出来高倍率
}

// DefaultTriangleFormationParams 標準パラメータ。
func DefaultTriangleFormationParams() TriangleFormationParams {
	return TriangleFormationParams{
		Window:                60,
		ExtremaStrength:       2,
		MinExtremaPoints:      3,
		MinSlopeRatio:         decimal.RequireFromString("0.0005"),
		RangeContractionRatio: decimal.RequireFromString("0.80"),
		BreakLookback:         20,
		VolumeWindow:          20,
		VolumeMultiplier:      decimal.RequireFromString("1.5"),
	}
}

// TriangleFormationEntrySignals 直近60営業日で高値切り下げ・安値切り上げのレンジ収縮が続いた後、
// 直近20日高値を当日ブレイクし出来高が伴った瞬間を検出する（DefaultTriangleFormationParams を使用）。
func TriangleFormationEntrySignals(prices []*models.StockBrandDailyPrice) []bool {
	return TriangleFormationEntrySignalsWithParams(prices, DefaultTriangleFormationParams())
}

// TriangleFormationEntrySignalsWithParams パラメータを指定して三角持ち合いブレイクを検出する。
// 判定は「前日までの窓(Window本)で収縮していた」かつ「当日ブレイクした」の AND。
// 収縮判定とブレイク判定の窓を分けることで、ブレイク当日のレンジ拡大が収縮判定を汚さないようにしている。
func TriangleFormationEntrySignalsWithParams(prices []*models.StockBrandDailyPrice, p TriangleFormationParams) []bool {
	n := len(prices)
	signals := make([]bool, n)
	if n == 0 {
		return signals
	}
	closes := ExtractClosePrices(prices)

	for i := p.Window; i < n; i++ {
		if i-1-p.BreakLookback < 0 {
			continue
		}

		// 1. 収縮窓（当日を含まない直近 Window 本）で高値切り下げ・安値切り上げを検証する。
		contractionWindow := prices[i-p.Window : i]
		highSlope, okH := localExtremaSlope(contractionWindow, true, p.ExtremaStrength, p.MinExtremaPoints)
		lowSlope, okL := localExtremaSlope(contractionWindow, false, p.ExtremaStrength, p.MinExtremaPoints)
		if !okH || !okL {
			continue
		}
		if closes[i].IsZero() {
			continue
		}
		highSlopeRatio := highSlope.Div(closes[i])
		lowSlopeRatio := lowSlope.Div(closes[i])
		if highSlopeRatio.GreaterThan(p.MinSlopeRatio.Neg()) {
			continue // 高値切り下げが十分でない
		}
		if lowSlopeRatio.LessThan(p.MinSlopeRatio) {
			continue // 安値切り上げが十分でない
		}

		// 2. レンジ収縮を直接検証する（極値の傾きだけでは日中レンジが縮んでいるとは限らないため）。
		third := p.Window / 3
		if third == 0 {
			continue
		}
		firstAvgRange := avgPriceRange(contractionWindow[:third])
		lastAvgRange := avgPriceRange(contractionWindow[len(contractionWindow)-third:])
		if firstAvgRange.IsZero() {
			continue
		}
		if lastAvgRange.Div(firstAvgRange).GreaterThan(p.RangeContractionRatio) {
			continue // レンジが収縮していない
		}

		// 3. 当日ブレイク（瞬間判定）: 当日終値が直近安値高値を上抜け、前日は未上抜け。
		prevHigh := maxHighInLookback(prices, i, p.BreakLookback)
		prevPrevHigh := maxHighInLookback(prices, i-1, p.BreakLookback)
		breakout := closes[i].GreaterThan(prevHigh) && !closes[i-1].GreaterThan(prevPrevHigh)
		if !breakout {
			continue
		}

		// 4. 出来高が直近平均を一定倍率以上上回っていること。
		if decimal.NewFromInt(prices[i].Volume).LessThanOrEqual(avgVolume(prices, i, p.VolumeWindow).Mul(p.VolumeMultiplier)) {
			continue
		}

		signals[i] = true
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
// 最小二乗回帰を当てて傾きを返す。極値は左右 strength 本より厳密に大小である点のみを採用する
// （fractal強度。同値は極値としない）。極値が minPoints 点未満なら ok=false。
func localExtremaSlope(window []*models.StockBrandDailyPrice, high bool, strength, minPoints int) (decimal.Decimal, bool) {
	type point struct {
		x int
		y decimal.Decimal
	}
	valueAt := func(idx int) decimal.Decimal {
		if high {
			return window[idx].High
		}
		return window[idx].Low
	}

	var pts []point
	for j := strength; j < len(window)-strength; j++ {
		v := valueAt(j)
		isExtreme := true
		for k := j - strength; k <= j+strength; k++ {
			if k == j {
				continue
			}
			nv := valueAt(k)
			if high {
				if nv.GreaterThanOrEqual(v) {
					isExtreme = false
					break
				}
			} else {
				if nv.LessThanOrEqual(v) {
					isExtreme = false
					break
				}
			}
		}
		if isExtreme {
			pts = append(pts, point{j, v})
		}
	}
	if len(pts) < minPoints {
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

// avgPriceRange 区間の平均日中レンジ (High-Low)。
func avgPriceRange(prices []*models.StockBrandDailyPrice) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, p := range prices {
		sum = sum.Add(p.High.Sub(p.Low))
	}
	return sum.Div(decimal.NewFromInt(int64(len(prices))))
}

// maxHighInLookback idx 直前 lookback 日（idx は含まない）の最大高値。範囲が空なら decimal.Zero。
func maxHighInLookback(prices []*models.StockBrandDailyPrice, idx, lookback int) decimal.Decimal {
	start := idx - lookback
	if start < 0 {
		start = 0
	}
	if start >= idx {
		return decimal.Zero
	}
	m := prices[start].High
	for j := start + 1; j < idx; j++ {
		if prices[j].High.GreaterThan(m) {
			m = prices[j].High
		}
	}
	return m
}

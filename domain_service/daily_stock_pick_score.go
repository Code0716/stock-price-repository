package domain_service

import (
	"github.com/shopspring/decimal"
)

// DailyPickScoreVersion スコア定義のバージョン。重み・カーブを変えたら必ずインクリメントし、
// DB の score_version と突き合わせて過去分と混ぜて集計しないようにする。
const DailyPickScoreVersion = "v1"

// DailyPickBaseStrategies スコアの点灯数カウント対象。
// multiple_signals は他4戦略の派生（2つ以上成立）であり、二重計上と計算の無駄を避けるため除外する。
var DailyPickBaseStrategies = []string{
	StrategyMACDBullish,
	StrategyBollingerBreakout,
	StrategyTriangleFormation,
	StrategyMovingAverageCross,
}

// DailyPickScoreWeights 各因子の重み。合計 100 になるよう保つ（ScoreDailyPick はスケーリングしない）。
type DailyPickScoreWeights struct {
	Signal     decimal.Decimal
	Volume     decimal.Decimal
	Trend      decimal.Decimal
	Volatility decimal.Decimal
	Liquidity  decimal.Decimal
	Overheat   decimal.Decimal
}

// DefaultDailyPickScoreWeights 標準の重み配分（合計100）。
func DefaultDailyPickScoreWeights() DailyPickScoreWeights {
	return DailyPickScoreWeights{
		Signal:     decimal.RequireFromString("30"),
		Volume:     decimal.RequireFromString("20"),
		Trend:      decimal.RequireFromString("20"),
		Volatility: decimal.RequireFromString("10"),
		Liquidity:  decimal.RequireFromString("10"),
		Overheat:   decimal.RequireFromString("10"),
	}
}

// scoreBreakpoint 因子の正規化カーブの折れ点。X=生値、Y=0..1の正規化値。
type scoreBreakpoint struct {
	X decimal.Decimal
	Y decimal.Decimal
}

// bp scoreBreakpoint を文字列から構築するヘルパー（decimal.NewFromFloat の誤差を避ける）。
func bp(x, y string) scoreBreakpoint {
	return scoreBreakpoint{X: decimal.RequireFromString(x), Y: decimal.RequireFromString(y)}
}

// normalizeByBreakpoints 折れ線補間で生値を 0..1 に正規化する。points は X 昇順・2点以上。
// 範囲外は端点の Y にクランプする。同一 X が連続する場合はゼロ除算を避けて後者の Y を採用する。
func normalizeByBreakpoints(v decimal.Decimal, points []scoreBreakpoint) decimal.Decimal {
	if len(points) == 0 {
		return decimal.Zero
	}
	if v.LessThanOrEqual(points[0].X) {
		return points[0].Y
	}
	last := points[len(points)-1]
	if v.GreaterThanOrEqual(last.X) {
		return last.Y
	}
	for i := 0; i < len(points)-1; i++ {
		p0, p1 := points[i], points[i+1]
		if v.GreaterThanOrEqual(p0.X) && v.LessThanOrEqual(p1.X) {
			denom := p1.X.Sub(p0.X)
			if denom.IsZero() {
				return p1.Y
			}
			ratio := v.Sub(p0.X).Div(denom)
			return p0.Y.Add(p1.Y.Sub(p0.Y).Mul(ratio))
		}
	}
	return last.Y
}

// 各因子の正規化カーブ。チューニングはここだけ変更すればよい。
var (
	// dailyPickSignalCurve 点灯戦略数: 1→0.30, 2→0.65, 3以上→1.00。
	dailyPickSignalCurve = []scoreBreakpoint{
		bp("1", "0.30"),
		bp("2", "0.65"),
		bp("3", "1.00"),
		bp("4", "1.00"),
	}
	// dailyPickVolumeCurve 出来高倍率（当日/直近平均）: 平均並み→0, 1.5倍→0.40, 2倍→0.70, 3倍以上→1.00。
	dailyPickVolumeCurve = []scoreBreakpoint{
		bp("1.0", "0"),
		bp("1.5", "0.40"),
		bp("2.0", "0.70"),
		bp("3.0", "1.00"),
	}
	// dailyPickADXCurve ADX(14): 15以下→0（レンジ）, 20→0.35, 30→0.80, 40以上→1.00。
	dailyPickADXCurve = []scoreBreakpoint{
		bp("15", "0"),
		bp("20", "0.35"),
		bp("30", "0.80"),
		bp("40", "1.00"),
	}
	// dailyPickATRRatioCurve ATR比率（台形）: 1%以下→0, 2〜5%→1.00, 10%以上→0（値動きが荒すぎる）。
	dailyPickATRRatioCurve = []scoreBreakpoint{
		bp("0.010", "0"),
		bp("0.020", "1.00"),
		bp("0.050", "1.00"),
		bp("0.100", "0"),
	}
	// dailyPickLiquidityCurve 直近平均売買代金: 1億→0, 3億→0.40, 10億→0.75, 50億以上→1.00。
	dailyPickLiquidityCurve = []scoreBreakpoint{
		bp("100000000", "0"),
		bp("300000000", "0.40"),
		bp("1000000000", "0.75"),
		bp("5000000000", "1.00"),
	}
	// dailyPickRSICurve RSI(14)（山型）: 30→0.40, 45→0.90, 60→1.00, 70→0.50, 80→0.10, 90以上→0。
	dailyPickRSICurve = []scoreBreakpoint{
		bp("30", "0.40"),
		bp("45", "0.90"),
		bp("60", "1.00"),
		bp("70", "0.50"),
		bp("80", "0.10"),
		bp("90", "0"),
	}
)

// ScoreDailyPick 各因子の生値から 0..100 の複合スコアを算出する（Round(2)済み）。
func ScoreDailyPick(m DailyPickMetrics, w DailyPickScoreWeights) decimal.Decimal {
	signal := normalizeByBreakpoints(decimal.NewFromInt(int64(m.SignalCount)), dailyPickSignalCurve)
	volume := normalizeByBreakpoints(m.VolumeRatio, dailyPickVolumeCurve)
	trend := normalizeByBreakpoints(m.ADX, dailyPickADXCurve)
	volatility := normalizeByBreakpoints(m.ATRRatio, dailyPickATRRatioCurve)
	liquidity := normalizeByBreakpoints(m.AvgTradingValue, dailyPickLiquidityCurve)
	overheat := normalizeByBreakpoints(m.RSI, dailyPickRSICurve)

	score := w.Signal.Mul(signal).
		Add(w.Volume.Mul(volume)).
		Add(w.Trend.Mul(trend)).
		Add(w.Volatility.Mul(volatility)).
		Add(w.Liquidity.Mul(liquidity)).
		Add(w.Overheat.Mul(overheat))

	return score.Round(2)
}

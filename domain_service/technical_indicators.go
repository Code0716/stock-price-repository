package domain_service

import (
	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
)

// 本ファイルのテクニカル指標は stt-golang(domain_service/technical_indicators.go)
// から移植した純粋関数。decimalSqrt は return_analytics.go の実装を同パッケージで共有する。

// ExtractClosePrices 日足スライスから終値（Close）のリストを抽出する。
// 既存戦略はいずれも終値で判定するため、シグナル算出はこの終値系列を用いる。
func ExtractClosePrices(dailyPrices []*models.StockBrandDailyPrice) []decimal.Decimal {
	prices := make([]decimal.Decimal, 0, len(dailyPrices))
	for _, p := range dailyPrices {
		prices = append(prices, p.Close)
	}
	return prices
}

// CalculateEMA 指数平滑移動平均 (EMA) を計算する。
func CalculateEMA(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) < period {
		return nil
	}

	ema := make([]decimal.Decimal, len(prices))
	// k = 2 / (period + 1)
	k := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(period + 1)))

	// 初回のEMAはSMA（単純移動平均）で代用する
	sum := decimal.Zero
	for i := 0; i < period; i++ {
		sum = sum.Add(prices[i])
	}
	ema[period-1] = sum.Div(decimal.NewFromInt(int64(period)))

	for i := period; i < len(prices); i++ {
		// EMA = (Close - PreviousEMA) * k + PreviousEMA
		ema[i] = prices[i].Sub(ema[i-1]).Mul(k).Add(ema[i-1])
	}

	return ema
}

// MACDResult MACDの計算結果
type MACDResult struct {
	MACD      decimal.Decimal
	Signal    decimal.Decimal
	Histogram decimal.Decimal
}

// CalculateMACD MACDを計算する (Short: 12, Long: 26, Signal: 9 が一般的)。
func CalculateMACD(prices []decimal.Decimal, shortPeriod, longPeriod, signalPeriod int) []MACDResult {
	if len(prices) < longPeriod+signalPeriod {
		return nil
	}

	shortEMA := CalculateEMA(prices, shortPeriod)
	longEMA := CalculateEMA(prices, longPeriod)

	macdLine := make([]decimal.Decimal, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < longPeriod-1 {
			macdLine[i] = decimal.Zero
			continue
		}
		macdLine[i] = shortEMA[i].Sub(longEMA[i])
	}

	// Signal Line は MACD Line の EMA。MACD Line の有効データは longPeriod-1 から。
	validMACDStart := longPeriod - 1
	if validMACDStart >= len(macdLine) {
		return nil
	}

	targetMACD := macdLine[validMACDStart:]
	signalEMA := CalculateEMA(targetMACD, signalPeriod)

	results := make([]MACDResult, len(prices))

	for j := 0; j < len(signalEMA); j++ {
		originalIndex := validMACDStart + j
		if j < signalPeriod-1 {
			continue
		}
		results[originalIndex] = MACDResult{
			MACD:      macdLine[originalIndex],
			Signal:    signalEMA[j],
			Histogram: macdLine[originalIndex].Sub(signalEMA[j]),
		}
	}

	return results
}

// CalculateRSI RSI (Relative Strength Index) を計算する (Wilder's Smoothing)。
func CalculateRSI(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) <= period {
		return nil
	}

	rsi := make([]decimal.Decimal, len(prices))

	gains := make([]decimal.Decimal, len(prices))
	losses := make([]decimal.Decimal, len(prices))

	for i := 1; i < len(prices); i++ {
		diff := prices[i].Sub(prices[i-1])
		if diff.GreaterThan(decimal.Zero) {
			gains[i] = diff
			losses[i] = decimal.Zero
		} else {
			gains[i] = decimal.Zero
			losses[i] = diff.Abs()
		}
	}

	avgGain := decimal.Zero
	avgLoss := decimal.Zero
	for i := 1; i <= period; i++ {
		avgGain = avgGain.Add(gains[i])
		avgLoss = avgLoss.Add(losses[i])
	}
	avgGain = avgGain.Div(decimal.NewFromInt(int64(period)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(period)))

	if avgLoss.IsZero() {
		rsi[period] = decimal.NewFromInt(100)
	} else {
		rs := avgGain.Div(avgLoss)
		rsi[period] = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
	}

	periodDec := decimal.NewFromInt(int64(period))
	periodMinusOne := decimal.NewFromInt(int64(period - 1))

	for i := period + 1; i < len(prices); i++ {
		avgGain = avgGain.Mul(periodMinusOne).Add(gains[i]).Div(periodDec)
		avgLoss = avgLoss.Mul(periodMinusOne).Add(losses[i]).Div(periodDec)

		if avgLoss.IsZero() {
			rsi[i] = decimal.NewFromInt(100)
		} else {
			rs := avgGain.Div(avgLoss)
			rsi[i] = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
		}
	}

	return rsi
}

// BollingerBandsResult ボリンジャーバンドの計算結果
type BollingerBandsResult struct {
	Upper     decimal.Decimal
	Middle    decimal.Decimal
	Lower     decimal.Decimal
	BandWidth decimal.Decimal // (Upper - Lower) / Middle
}

// CalculateBollingerBands ボリンジャーバンドを計算する (SMAベース)。
func CalculateBollingerBands(prices []decimal.Decimal, period int, multiplier decimal.Decimal) []BollingerBandsResult {
	if len(prices) < period {
		return nil
	}

	results := make([]BollingerBandsResult, len(prices))

	for i := period - 1; i < len(prices); i++ {
		sum := decimal.Zero
		for j := i - period + 1; j <= i; j++ {
			sum = sum.Add(prices[j])
		}
		sma := sum.Div(decimal.NewFromInt(int64(period)))

		variance := decimal.Zero
		for j := i - period + 1; j <= i; j++ {
			diff := prices[j].Sub(sma)
			variance = variance.Add(diff.Mul(diff))
		}
		variance = variance.Div(decimal.NewFromInt(int64(period)))
		stdDev := decimalSqrt(variance)

		upper := sma.Add(multiplier.Mul(stdDev))
		lower := sma.Sub(multiplier.Mul(stdDev))

		bandWidth := decimal.Zero
		if !sma.IsZero() {
			bandWidth = upper.Sub(lower).Div(sma)
		}

		results[i] = BollingerBandsResult{
			Upper:     upper,
			Middle:    sma,
			Lower:     lower,
			BandWidth: bandWidth,
		}
	}

	return results
}

// decimalMax 2つの decimal のうち大きい方を返す。
func decimalMax(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

// trueRange 当日の High/Low と前日終値から True Range を返す。
// TR = max(H-L, |H-prevClose|, |L-prevClose|)
func trueRange(high, low, prevClose decimal.Decimal) decimal.Decimal {
	hl := high.Sub(low)
	hpc := high.Sub(prevClose).Abs()
	lpc := low.Sub(prevClose).Abs()
	return decimalMax(hl, decimalMax(hpc, lpc))
}

// CalculateATR ATR (Average True Range) を Wilder 平滑で計算する（既定 period=14）。
// 戻り値長は len(prices)。index < period は未確定で Zero。
func CalculateATR(prices []*models.StockBrandDailyPrice, period int) []decimal.Decimal {
	n := len(prices)
	if period <= 0 || n <= period {
		return nil
	}

	tr := make([]decimal.Decimal, n)
	for i := 1; i < n; i++ {
		tr[i] = trueRange(prices[i].High, prices[i].Low, prices[i-1].Close)
	}

	periodDec := decimal.NewFromInt(int64(period))
	periodMinusOne := decimal.NewFromInt(int64(period - 1))

	atr := make([]decimal.Decimal, n)
	sum := decimal.Zero
	for i := 1; i <= period; i++ {
		sum = sum.Add(tr[i])
	}
	atr[period] = sum.Div(periodDec)

	for i := period + 1; i < n; i++ {
		atr[i] = atr[i-1].Mul(periodMinusOne).Add(tr[i]).Div(periodDec)
	}

	return atr
}

// StochasticsResult ストキャスティクスの計算結果（%K, %D）。
type StochasticsResult struct {
	K decimal.Decimal
	D decimal.Decimal
}

// CalculateStochastics ストキャスティクス（Fast %K と その SMA の %D）を計算する。
// %K = (Close - 期間安値) / (期間高値 - 期間安値) * 100（既定 kPeriod=14, dPeriod=3）。
// レンジがゼロ（横ばい）の場合はゼロ除算回避のため %K=0 とする。
func CalculateStochastics(prices []*models.StockBrandDailyPrice, kPeriod, dPeriod int) []StochasticsResult {
	n := len(prices)
	if kPeriod <= 0 || dPeriod <= 0 || n < kPeriod+dPeriod-1 {
		return nil
	}

	hundred := decimal.NewFromInt(100)
	kValues := make([]decimal.Decimal, n)
	results := make([]StochasticsResult, n)

	for i := kPeriod - 1; i < n; i++ {
		hh := prices[i].High
		ll := prices[i].Low
		for j := i - kPeriod + 1; j <= i; j++ {
			if prices[j].High.GreaterThan(hh) {
				hh = prices[j].High
			}
			if prices[j].Low.LessThan(ll) {
				ll = prices[j].Low
			}
		}
		rng := hh.Sub(ll)
		if !rng.IsZero() {
			kValues[i] = prices[i].Close.Sub(ll).Div(rng).Mul(hundred)
		}
		results[i].K = kValues[i]
	}

	firstK := kPeriod - 1
	dPeriodDec := decimal.NewFromInt(int64(dPeriod))
	for i := firstK + dPeriod - 1; i < n; i++ {
		sum := decimal.Zero
		for j := i - dPeriod + 1; j <= i; j++ {
			sum = sum.Add(kValues[j])
		}
		results[i].D = sum.Div(dPeriodDec)
	}

	return results
}

// ADXResult ADX/DMI の計算結果（ADX と +DI / -DI）。
type ADXResult struct {
	ADX     decimal.Decimal
	PlusDI  decimal.Decimal
	MinusDI decimal.Decimal
}

// calcDMTR 各バーの +DM / -DM / TR を返す（CalculateADX の補助関数）。
func calcDMTR(prices []*models.StockBrandDailyPrice) (plusDM, minusDM, tr []decimal.Decimal) {
	n := len(prices)
	plusDM = make([]decimal.Decimal, n)
	minusDM = make([]decimal.Decimal, n)
	tr = make([]decimal.Decimal, n)
	for i := 1; i < n; i++ {
		upMove := prices[i].High.Sub(prices[i-1].High)
		downMove := prices[i-1].Low.Sub(prices[i].Low)
		if upMove.GreaterThan(downMove) && upMove.GreaterThan(decimal.Zero) {
			plusDM[i] = upMove
		}
		if downMove.GreaterThan(upMove) && downMove.GreaterThan(decimal.Zero) {
			minusDM[i] = downMove
		}
		tr[i] = trueRange(prices[i].High, prices[i].Low, prices[i-1].Close)
	}
	return
}

// CalculateADX ADX（平均方向性指数）と +DI / -DI を Wilder 方式で計算する（既定 period=14）。
// +DI/-DI は index>=period、ADX は index>=2*period-1 から確定する。
func CalculateADX(prices []*models.StockBrandDailyPrice, period int) []ADXResult {
	n := len(prices)
	if period <= 0 || n < 2*period {
		return nil
	}

	plusDM, minusDM, tr := calcDMTR(prices)

	periodDec := decimal.NewFromInt(int64(period))
	smPlus := make([]decimal.Decimal, n)
	smMinus := make([]decimal.Decimal, n)
	smTR := make([]decimal.Decimal, n)
	for i := 1; i <= period; i++ {
		smPlus[period] = smPlus[period].Add(plusDM[i])
		smMinus[period] = smMinus[period].Add(minusDM[i])
		smTR[period] = smTR[period].Add(tr[i])
	}
	for i := period + 1; i < n; i++ {
		smPlus[i] = smPlus[i-1].Sub(smPlus[i-1].Div(periodDec)).Add(plusDM[i])
		smMinus[i] = smMinus[i-1].Sub(smMinus[i-1].Div(periodDec)).Add(minusDM[i])
		smTR[i] = smTR[i-1].Sub(smTR[i-1].Div(periodDec)).Add(tr[i])
	}

	hundred := decimal.NewFromInt(100)
	results := make([]ADXResult, n)
	dx := make([]decimal.Decimal, n)
	for i := period; i < n; i++ {
		if smTR[i].IsZero() {
			continue
		}
		pdi := smPlus[i].Div(smTR[i]).Mul(hundred)
		mdi := smMinus[i].Div(smTR[i]).Mul(hundred)
		results[i].PlusDI = pdi
		results[i].MinusDI = mdi
		if diSum := pdi.Add(mdi); !diSum.IsZero() {
			dx[i] = pdi.Sub(mdi).Abs().Div(diSum).Mul(hundred)
		}
	}

	adxStart := 2*period - 1
	if adxStart >= n {
		return results
	}
	var dsum decimal.Decimal
	for i := period; i <= adxStart; i++ {
		dsum = dsum.Add(dx[i])
	}
	results[adxStart].ADX = dsum.Div(periodDec)
	periodMinusOne := decimal.NewFromInt(int64(period - 1))
	for i := adxStart + 1; i < n; i++ {
		results[i].ADX = results[i-1].ADX.Mul(periodMinusOne).Add(dx[i]).Div(periodDec)
	}

	return results
}

// CalculateOBV OBV (On-Balance Volume) を計算する。
// 終値が前日比上昇なら出来高を加算、下落なら減算、変わらずなら据え置き。index 0 は基準 0。
func CalculateOBV(prices []*models.StockBrandDailyPrice) []decimal.Decimal {
	n := len(prices)
	if n == 0 {
		return nil
	}

	obv := make([]decimal.Decimal, n)
	for i := 1; i < n; i++ {
		vol := decimal.NewFromInt(prices[i].Volume)
		switch {
		case prices[i].Close.GreaterThan(prices[i-1].Close):
			obv[i] = obv[i-1].Add(vol)
		case prices[i].Close.LessThan(prices[i-1].Close):
			obv[i] = obv[i-1].Sub(vol)
		default:
			obv[i] = obv[i-1]
		}
	}

	return obv
}

// CalculateRollingVWAP 期間 period の移動 VWAP を計算する（既定 period=14）。
// 典型価格 (H+L+C)/3 を出来高加重平均する。index < period-1 は未確定で Zero。
// 期間内の出来高合計がゼロの場合はゼロ除算回避のため Zero のままとする。
func CalculateRollingVWAP(prices []*models.StockBrandDailyPrice, period int) []decimal.Decimal {
	n := len(prices)
	if period <= 0 || n < period {
		return nil
	}

	three := decimal.NewFromInt(3)
	vwap := make([]decimal.Decimal, n)
	for i := period - 1; i < n; i++ {
		var pv, vv decimal.Decimal
		for j := i - period + 1; j <= i; j++ {
			typical := prices[j].High.Add(prices[j].Low).Add(prices[j].Close).Div(three)
			vol := decimal.NewFromInt(prices[j].Volume)
			pv = pv.Add(typical.Mul(vol))
			vv = vv.Add(vol)
		}
		if vv.IsZero() {
			continue
		}
		vwap[i] = pv.Div(vv)
	}

	return vwap
}

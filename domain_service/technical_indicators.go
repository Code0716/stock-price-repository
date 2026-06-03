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

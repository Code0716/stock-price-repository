package daytrade

import (
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// DailyLowHighKey は銘柄×日で当日の安値・高値を引くためのキー。
type DailyLowHighKey struct {
	TickerSymbol string
	ExecutedOn   string // YYYY-MM-DD
}

// DailyLowHigh は当日の安値・高値（生値。分割調整前の実際の呼び値を使うこと）。
type DailyLowHigh struct {
	Low  decimal.Decimal
	High decimal.Decimal
}

// ComputeStopCompliance は宣言済み損切りライン（daytrade_trade_notes.declared_stop_price）と
// 当日の安値・高値を突き合わせ、宣言を守れたか破ったかを分類する。
//
// 建玉の売買方向（買い/売り）はSBI CSVの取引区分表記（現物買・信用返済売 等）に依存せず、
// 「宣言ストップ価格が数量加重平均取得単価より低いか高いか」から推定する。
// ストップが取得単価より低ければ買い建玉（ロング）、高ければ売り建玉（ショート）とみなす。
// 一致する場合や数量・当日日足が取れない場合は unknown として除外する。
//
// 同一バーで損切り・利確の両方に到達したかは日足からは判別できないため、この関数は
// 「到達したかどうか」と「その日の実損益」だけで分類する（ザラ場の経路は復元しない）。
func ComputeStopCompliance(
	trades []*models.DaytradeTradeApprox,
	notes []*models.DaytradeTradeNoteRecord,
	priceByKey map[DailyLowHighKey]DailyLowHigh,
) *models.DaytradeStopCompliance {
	noteMap := make(map[tradeNoteKey]*models.DaytradeTradeNoteRecord, len(notes))
	for _, n := range notes {
		if n.DeclaredStopPrice == nil {
			continue
		}
		k := tradeNoteKey{
			tickerSymbol: n.TickerSymbol,
			executedOn:   n.ExecutedOn.Format("2006-01-02"),
			direction:    n.Direction,
		}
		noteMap[k] = n
	}

	result := &models.DaytradeStopCompliance{}

	for _, t := range trades {
		executedOn := t.ExecutedOn.Format("2006-01-02")
		k := tradeNoteKey{
			tickerSymbol: t.TickerSymbol,
			executedOn:   executedOn,
			direction:    t.Direction,
		}
		n, ok := noteMap[k]
		if !ok {
			continue
		}

		stop := *n.DeclaredStopPrice
		result.DeclaredCount++

		row := models.DaytradeStopComplianceTrade{
			TickerSymbol:        t.TickerSymbol,
			BrandName:           t.BrandName,
			ExecutedOn:          executedOn,
			Direction:           t.Direction,
			DeclaredStopPrice:   stop,
			AvgAcquisitionPrice: t.AvgAcquisitionPrice,
			Quantity:            t.Quantity,
			ProfitLoss:          t.ProfitLoss,
		}

		prices, hasPrices := priceByKey[DailyLowHighKey{TickerSymbol: t.TickerSymbol, ExecutedOn: executedOn}]

		switch {
		case t.Quantity == 0 || t.AvgAcquisitionPrice.IsZero() || stop.Equal(t.AvgAcquisitionPrice) || !hasPrices:
			row.Category = models.DaytradeStopComplianceUnknown
			result.UnknownCount++
		default:
			classifyTouchedStop(&row, t, stop, prices, result)
		}

		result.Trades = append(result.Trades, row)
	}

	if result.BreachedWorseCount > 0 {
		result.AvgOvershoot = float64(result.TotalOvershoot) / float64(result.BreachedWorseCount)
	}

	return result
}

// classifyTouchedStop は方向判定・到達判定・到達後の結果分類を行い row と result を更新する。
func classifyTouchedStop(
	row *models.DaytradeStopComplianceTrade,
	t *models.DaytradeTradeApprox,
	stop decimal.Decimal,
	prices DailyLowHigh,
	result *models.DaytradeStopCompliance,
) {
	low, high := prices.Low, prices.High
	row.DayLow = &low
	row.DayHigh = &high

	isLong := stop.LessThan(t.AvgAcquisitionPrice)
	touched := high.GreaterThanOrEqual(stop)
	if isLong {
		touched = low.LessThanOrEqual(stop)
	}

	if !touched {
		row.Category = models.DaytradeStopComplianceNotTriggered
		result.NotTriggeredCount++
		return
	}

	// 宣言通りに手仕舞えていた場合の想定損失（円）。ロングは 取得単価-ストップ、ショートは ストップ-取得単価。
	diff := t.AvgAcquisitionPrice.Sub(stop)
	if !isLong {
		diff = stop.Sub(t.AvgAcquisitionPrice)
	}
	expectedLoss := diff.Mul(decimal.NewFromInt(int64(t.Quantity))).Round(0).IntPart()
	row.ExpectedLossAtStop = &expectedLoss

	if t.ProfitLoss >= 0 {
		// 到達したのに手仕舞わず持ち越し、建値/利益で終えた＝宣言を破ったが結果的に逃げ切った
		row.Category = models.DaytradeStopComplianceBreachedRecovered
		result.BreachedRecoveredCount++
		return
	}

	actualLoss := -t.ProfitLoss
	overshoot := actualLoss - expectedLoss
	if overshoot > 0 {
		row.Category = models.DaytradeStopComplianceBreachedWorse
		result.BreachedWorseCount++
		result.TotalOvershoot += overshoot
		row.Overshoot = &overshoot
		return
	}

	row.Category = models.DaytradeStopComplianceHonored
	result.HonoredCount++
}

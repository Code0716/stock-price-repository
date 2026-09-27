package daytrade

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// feeAnomalyThreshold を超える fee は異常値として扱う（本来 fee は0以下のはず）。
const feeAnomalyThreshold = 1

// ExecutionFee は約定1行の値洗い前損益（gross）と手数料相当額（fee）を算出する。
// 返済買（ショート決済）: gross = (averageCost - unitPrice) * quantity
// それ以外（返済売などロング決済）: gross = (unitPrice - averageCost) * quantity
// gross は1円単位に四捨五入し、fee = profitLoss - gross。|fee| <= feeAnomalyThreshold は丸め誤差として0扱いにする。
func ExecutionFee(ex *models.DaytradeExecution) (gross, fee int64) {
	qty := decimal.NewFromInt(int64(ex.Quantity))

	var grossDec decimal.Decimal
	switch ex.MarginKind {
	case "返済買":
		grossDec = ex.AverageCost.Sub(ex.UnitPrice).Mul(qty)
	default:
		grossDec = ex.UnitPrice.Sub(ex.AverageCost).Mul(qty)
	}
	gross = grossDec.Round(0).IntPart()

	fee = ex.ProfitLoss - gross
	if fee >= -feeAnomalyThreshold && fee <= feeAnomalyThreshold {
		fee = 0
	}
	return gross, fee
}

// IsFeeAnomaly は fee がプラス（本来ありえない）かどうかを返す。
func IsFeeAnomaly(fee int64) bool {
	return fee > feeAnomalyThreshold
}

// FeeByBucket は executions を granularity に応じたバケットキーで集計した手数料合計を返す。
// キーの形式は repositories.DaytradeExecutionRepository.Aggregate が返す bucket_date と揃える
// (daily=YYYY-MM-DD, monthly=YYYY-MM-01, yearly=YYYY-01-01, all=空文字)。
func FeeByBucket(executions []*models.DaytradeExecution, g models.DaytradeSummaryGranularity) map[string]int64 {
	result := make(map[string]int64)
	for _, ex := range executions {
		_, fee := ExecutionFee(ex)
		if fee == 0 {
			continue
		}
		var key string
		switch g {
		case models.DaytradeSummaryGranularityDaily:
			key = ex.ExecutedOn.Format("2006-01-02")
		case models.DaytradeSummaryGranularityMonthly:
			key = ex.ExecutedOn.Format("2006-01") + "-01"
		case models.DaytradeSummaryGranularityYearly:
			key = ex.ExecutedOn.Format("2006") + "-01-01"
		case models.DaytradeSummaryGranularityAll:
			key = ""
		}
		result[key] += fee
	}
	return result
}

// ApplyFee は約定1行に Fee / GrossProfitLoss を設定する。
func ApplyFee(ex *models.DaytradeExecution) {
	_, fee := ExecutionFee(ex)
	ex.Fee = fee
	ex.GrossProfitLoss = ex.ProfitLoss - fee
}

type feeMonthlyAccum struct {
	fee             int64
	profitLoss      int64
	grossProfitLoss int64
}

type feeDayAccum struct {
	executedOn   string
	tickerSymbol string
	brandName    string
	fee          int64
}

// BuildFeeReport は約定明細から手数料レポートを構築する。
// executions は Fee 未計算でもよい（内部で ExecutionFee を都度計算する）。
func BuildFeeReport(executions []*models.DaytradeExecution) *models.DaytradeFeeReport {
	report := &models.DaytradeFeeReport{
		Monthly:   make([]models.DaytradeFeeMonthly, 0),
		FeeDays:   make([]models.DaytradeFeeDay, 0),
		Anomalies: make([]models.DaytradeFeeAnomaly, 0),
	}

	monthOrder := make([]string, 0)
	monthAcc := make(map[string]*feeMonthlyAccum)

	dayKeyOrder := make([]string, 0)
	dayAcc := make(map[string]*feeDayAccum)

	for _, ex := range executions {
		gross, fee := ExecutionFee(ex)
		grossProfitLoss := ex.ProfitLoss - fee

		report.ProfitLoss += ex.ProfitLoss
		report.GrossProfitLoss += grossProfitLoss

		month := ex.ExecutedOn.Format("2006-01")
		if _, ok := monthAcc[month]; !ok {
			monthOrder = append(monthOrder, month)
			monthAcc[month] = &feeMonthlyAccum{}
		}
		ma := monthAcc[month]
		ma.fee += fee
		ma.profitLoss += ex.ProfitLoss
		ma.grossProfitLoss += grossProfitLoss

		if fee != 0 {
			report.TotalFee += fee
			report.FeeRowCount++

			dayKey := ex.ExecutedOn.Format("2006-01-02") + "|" + ex.TickerSymbol
			if _, ok := dayAcc[dayKey]; !ok {
				dayKeyOrder = append(dayKeyOrder, dayKey)
				dayAcc[dayKey] = &feeDayAccum{
					executedOn:   ex.ExecutedOn.Format("2006-01-02"),
					tickerSymbol: ex.TickerSymbol,
					brandName:    ex.BrandName,
				}
			}
			dayAcc[dayKey].fee += fee
		}

		if IsFeeAnomaly(fee) {
			report.Anomalies = append(report.Anomalies, models.DaytradeFeeAnomaly{
				ID:           ex.ID,
				ExecutedOn:   ex.ExecutedOn.Format("2006-01-02"),
				TickerSymbol: ex.TickerSymbol,
				BrandName:    ex.BrandName,
				MarginKind:   ex.MarginKind,
				Quantity:     ex.Quantity,
				UnitPrice:    ex.UnitPrice,
				AverageCost:  ex.AverageCost,
				ProfitLoss:   ex.ProfitLoss,
				Gross:        gross,
				Fee:          fee,
			})
		}
	}

	for _, m := range monthOrder {
		a := monthAcc[m]
		report.Monthly = append(report.Monthly, models.DaytradeFeeMonthly{
			Month:           m,
			Fee:             a.fee,
			ProfitLoss:      a.profitLoss,
			GrossProfitLoss: a.grossProfitLoss,
		})
	}

	dayKeySet := make(map[string]struct{}, len(dayKeyOrder))
	for _, k := range dayKeyOrder {
		a := dayAcc[k]
		report.FeeDays = append(report.FeeDays, models.DaytradeFeeDay{
			ExecutedOn:   a.executedOn,
			TickerSymbol: a.tickerSymbol,
			BrandName:    a.brandName,
			Fee:          a.fee,
		})
		dayKeySet[a.executedOn] = struct{}{}
	}
	report.FeeDayCount = len(dayKeySet)

	sort.Slice(report.Anomalies, func(i, j int) bool {
		return report.Anomalies[i].ExecutedOn < report.Anomalies[j].ExecutedOn
	})

	return report
}

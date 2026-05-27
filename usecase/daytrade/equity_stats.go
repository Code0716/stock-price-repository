package daytrade

import "github.com/Code0716/stock-price-repository/models"

// ComputeEquityStats は日次バケット（executed_on 昇順）から
// 最大ドローダウン（ピーク初期値 0）・最大ランアップ（谷初期値 0）・
// 最大連敗日数を計算する。いずれも正の金額。
//
// 約定日が日付精度のみのため、連敗はトレード単位ではなく「負け日の連続」として定義する。
func ComputeEquityStats(daily []*models.DaytradeSummaryBucket) (maxDrawdown, maxRunup int64, maxLossStreak int) {
	var cumulative int64
	var peak int64   // ピーク初期値 0（全損失なら初日からドローダウンとして計上）
	var trough int64 // 谷初期値 0（全利益なら初日からランアップとして計上）
	var streak int

	for _, b := range daily {
		cumulative += b.ProfitLoss

		if cumulative > peak {
			peak = cumulative
		}
		if dd := peak - cumulative; dd > maxDrawdown {
			maxDrawdown = dd
		}

		if cumulative < trough {
			trough = cumulative
		}
		if ru := cumulative - trough; ru > maxRunup {
			maxRunup = ru
		}

		if b.ProfitLoss < 0 {
			streak++
			if streak > maxLossStreak {
				maxLossStreak = streak
			}
		} else {
			streak = 0
		}
	}
	return maxDrawdown, maxRunup, maxLossStreak
}

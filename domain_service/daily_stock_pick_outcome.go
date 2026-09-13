package domain_service

import (
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// JudgeDailyPickOutcome 5営業日後リターンから勝敗を判定する。プラスなら win、マイナスなら lose、ゼロなら draw。
func JudgeDailyPickOutcome(return5D decimal.Decimal) models.DailyStockPickOutcome {
	switch {
	case return5D.IsPositive():
		return models.DailyStockPickOutcomeWin
	case return5D.IsNegative():
		return models.DailyStockPickOutcomeLose
	default:
		return models.DailyStockPickOutcomeDraw
	}
}

// SummarizeDailyStockPicks 答え合わせ済みの推奨群からサマリを作る（void は勝率の分母から除く）。
// 全件が未評価（EvaluatedAt==nil）または該当なしの場合は nil を返す。
func SummarizeDailyStockPicks(picks []*models.DailyStockPick) *models.DailyStockPickSummary {
	if len(picks) == 0 {
		return nil
	}

	s := &models.DailyStockPickSummary{
		PickDate: picks[0].PickDate,
		Total:    len(picks),
	}

	sum1D, cnt1D := decimal.Zero, 0
	sum5D, cnt5D := decimal.Zero, 0
	var best, worst *models.DailyStockPick

	for _, p := range picks {
		countDailyStockPickOutcome(s, p.Outcome)

		if p.Return1D != nil {
			sum1D = sum1D.Add(*p.Return1D)
			cnt1D++
		}
		if p.Return5D != nil {
			sum5D = sum5D.Add(*p.Return5D)
			cnt5D++
			if best == nil || p.Return5D.GreaterThan(*best.Return5D) {
				best = p
			}
			if worst == nil || p.Return5D.LessThan(*worst.Return5D) {
				worst = p
			}
		}
	}

	decided := s.Win + s.Lose + s.Draw
	if decided > 0 {
		s.WinRate = decimal.NewFromInt(int64(s.Win)).Div(decimal.NewFromInt(int64(decided))).Round(4)
	}
	if cnt1D > 0 {
		s.AvgReturn1D = sum1D.Div(decimal.NewFromInt(int64(cnt1D))).Round(6)
	}
	if cnt5D > 0 {
		s.AvgReturn5D = sum5D.Div(decimal.NewFromInt(int64(cnt5D))).Round(6)
	}
	s.Best = best
	s.Worst = worst

	return s
}

// countDailyStockPickOutcome outcome に応じてサマリの勝敗カウンタを1件加算する。
func countDailyStockPickOutcome(s *models.DailyStockPickSummary, outcome *models.DailyStockPickOutcome) {
	if outcome == nil {
		return
	}
	switch *outcome {
	case models.DailyStockPickOutcomeWin:
		s.Win++
	case models.DailyStockPickOutcomeLose:
		s.Lose++
	case models.DailyStockPickOutcomeDraw:
		s.Draw++
	case models.DailyStockPickOutcomeVoid:
		s.Void++
	}
}

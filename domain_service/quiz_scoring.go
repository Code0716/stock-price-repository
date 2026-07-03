package domain_service

import (
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// GradeQuizAnswer 翌営業日終値 vs 基準終値の方向で回答を採点する。
// 「強く」は回答者の確信度を表すため、当たれば加点・外れれば減点が通常より大きくなる
// （的中率だけでなく確信度の使い方も評価するキャリブレーション設計）。
//   - r == 0（変わらず）は draw、スコア0
//   - 方向一致: 通常 +1 / 強く +2
//   - 方向不一致: 通常 -1 / 強く -2
func GradeQuizAnswer(prediction models.QuizPrediction, baseClose, nextClose decimal.Decimal) (models.QuizOutcome, int, decimal.Decimal) {
	actualReturn := nextClose.Sub(baseClose).Div(baseClose)

	if actualReturn.IsZero() {
		return models.QuizOutcome(models.QuizOutcomeDraw), 0, actualReturn
	}

	actualUp := actualReturn.IsPositive()
	correct := prediction.IsUp() == actualUp

	score := 1
	if prediction.IsStrong() {
		score = 2
	}
	if !correct {
		score = -score
	}

	outcome := models.QuizOutcomeIncorrect
	if correct {
		outcome = models.QuizOutcomeCorrect
	}

	return outcome, score, actualReturn
}

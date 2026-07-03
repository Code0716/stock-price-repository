package domain_service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func TestGradeQuizAnswer(t *testing.T) {
	base := decimal.NewFromInt(100)

	tests := []struct {
		name        string
		prediction  models.QuizPrediction
		nextClose   decimal.Decimal
		wantOutcome models.QuizOutcome
		wantScore   int
	}{
		{"通常予想・上昇的中", models.QuizPredictionUp, decimal.NewFromInt(105), models.QuizOutcomeCorrect, 1},
		{"通常予想・下落外れ", models.QuizPredictionDown, decimal.NewFromInt(105), models.QuizOutcomeIncorrect, -1},
		{"強い予想・上昇的中は+2", models.QuizPredictionStrongUp, decimal.NewFromInt(105), models.QuizOutcomeCorrect, 2},
		{"強い予想・下落外れは-2", models.QuizPredictionStrongDown, decimal.NewFromInt(105), models.QuizOutcomeIncorrect, -2},
		{"通常予想・下落的中", models.QuizPredictionDown, decimal.NewFromInt(95), models.QuizOutcomeCorrect, 1},
		{"強い予想・下落的中は+2", models.QuizPredictionStrongDown, decimal.NewFromInt(95), models.QuizOutcomeCorrect, 2},
		{"変わらずはdraw・0点", models.QuizPredictionStrongUp, decimal.NewFromInt(100), models.QuizOutcomeDraw, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome, score, _ := GradeQuizAnswer(tt.prediction, base, tt.nextClose)
			assert.Equal(t, tt.wantOutcome, outcome)
			assert.Equal(t, tt.wantScore, score)
		})
	}
}

func TestGradeQuizAnswer_ActualReturn(t *testing.T) {
	_, _, actualReturn := GradeQuizAnswer(models.QuizPredictionUp, decimal.NewFromInt(100), decimal.NewFromInt(110))
	assert.True(t, actualReturn.Equal(decimal.NewFromFloat(0.1)))
}

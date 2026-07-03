package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// QuizPrediction クイズの回答（4択）。「強く」は回答者の確信度であり、値動きの大きさではない。
type QuizPrediction string

const (
	QuizPredictionStrongDown QuizPrediction = "strong_down"
	QuizPredictionDown       QuizPrediction = "down"
	QuizPredictionUp         QuizPrediction = "up"
	QuizPredictionStrongUp   QuizPrediction = "strong_up"
)

// Valid 回答値が4択のいずれかであるかを検証する。
func (p QuizPrediction) Valid() bool {
	switch p {
	case QuizPredictionStrongDown, QuizPredictionDown, QuizPredictionUp, QuizPredictionStrongUp:
		return true
	}
	return false
}

// IsUp 上がると予想しているか（強い/弱い問わず）。
func (p QuizPrediction) IsUp() bool {
	return p == QuizPredictionUp || p == QuizPredictionStrongUp
}

// IsStrong 強い確信度の回答か。
func (p QuizPrediction) IsStrong() bool {
	return p == QuizPredictionStrongDown || p == QuizPredictionStrongUp
}

// QuizOutcome 採点結果。
type QuizOutcome string

const (
	QuizOutcomeCorrect   QuizOutcome = "correct"
	QuizOutcomeIncorrect QuizOutcome = "incorrect"
	QuizOutcomeDraw      QuizOutcome = "draw"
	QuizOutcomeVoid      QuizOutcome = "void"
)

// QuizUniverseEntry quiz_daily_universe のドメインモデル（この行自体が「設問」を兼ねる）。
type QuizUniverseEntry struct {
	QuizDate        time.Time
	StockBrandID    string
	TickerSymbol    string
	QuestionOrder   int
	AvgTradingValue decimal.Decimal
	AvgDailyRange   decimal.Decimal
	BaseClosePrice  decimal.Decimal
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// QuizAnswer quiz_answer のドメインモデル。採点前は Next*/Outcome/Score は nil。
type QuizAnswer struct {
	ID             uint64
	QuizDate       time.Time
	StockBrandID   string
	TickerSymbol   string
	Prediction     QuizPrediction
	AnsweredAt     time.Time
	NextClosePrice *decimal.Decimal
	ActualReturn   *decimal.Decimal
	Outcome        *QuizOutcome
	Score          *int
	GradedAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Graded 採点済みかどうか。
func (a *QuizAnswer) Graded() bool {
	return a.Outcome != nil
}

// --- API レスポンス用の構造体 ---

// QuizQuestion 1銘柄分の設問（銘柄名・コードは含まない＝非公開）。
type QuizQuestion struct {
	StockBrandID  string  `json:"stockBrandId"`
	QuestionOrder int     `json:"questionOrder"`
	Answered      bool    `json:"answered"`
	Prediction    *string `json:"prediction,omitempty"`
}

// QuizQuestionSet GET /quiz/questions のレスポンス。
type QuizQuestionSet struct {
	QuizDate      string          `json:"quizDate"`
	TotalCount    int             `json:"totalCount"`
	AnsweredCount int             `json:"answeredCount"`
	Questions     []*QuizQuestion `json:"questions"`
}

// QuizChartCandle チャート用の1本の日足。
type QuizChartCandle struct {
	Date   string          `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume int64           `json:"volume"`
}

// QuizChartMAPoint 移動平均線の1点。
type QuizChartMAPoint struct {
	Date  string          `json:"date"`
	Value decimal.Decimal `json:"value"`
}

// QuizChart GET /quiz/chart のレスポンス（銘柄名・コードは含まない）。
type QuizChart struct {
	QuizDate string              `json:"quizDate"`
	Candles  []*QuizChartCandle  `json:"candles"`
	MA5      []*QuizChartMAPoint `json:"ma5"`
	MA25     []*QuizChartMAPoint `json:"ma25"`
	MA75     []*QuizChartMAPoint `json:"ma75"`
}

// QuizResultItem 採点済み1件の結果（銘柄名を公開）。
type QuizResultItem struct {
	QuestionOrder  int              `json:"questionOrder"`
	TickerSymbol   string           `json:"tickerSymbol"`
	Name           string           `json:"name"`
	Prediction     QuizPrediction   `json:"prediction"`
	BaseClosePrice decimal.Decimal  `json:"baseClosePrice"`
	NextClosePrice *decimal.Decimal `json:"nextClosePrice"`
	ActualReturn   *decimal.Decimal `json:"actualReturn"`
	Outcome        *QuizOutcome     `json:"outcome"`
	Score          *int             `json:"score"`
}

// QuizResultsSummary 日別の採点サマリー。
type QuizResultsSummary struct {
	Answered  int `json:"answered"`
	Correct   int `json:"correct"`
	Incorrect int `json:"incorrect"`
	Draw      int `json:"draw"`
	Void      int `json:"void"`
	Score     int `json:"score"`
}

// QuizResults GET /quiz/results のレスポンス。
type QuizResults struct {
	QuizDate string             `json:"quizDate"`
	Graded   bool               `json:"graded"`
	Summary  QuizResultsSummary `json:"summary"`
	Items    []*QuizResultItem  `json:"items"`
}

// QuizConfidenceStats 確信度別（normal/strong）の的中統計。
type QuizConfidenceStats struct {
	Answered int             `json:"answered"`
	Correct  int             `json:"correct"`
	Accuracy decimal.Decimal `json:"accuracy"`
}

// QuizDailyScore 日別スコア推移の1点。
type QuizDailyScore struct {
	QuizDate string `json:"quizDate"`
	Answered int    `json:"answered"`
	Correct  int    `json:"correct"`
	Score    int    `json:"score"`
}

// QuizStats GET /quiz/stats のレスポンス。
type QuizStats struct {
	TotalScore    int                   `json:"totalScore"`
	TotalAnswered int                   `json:"totalAnswered"`
	TotalCorrect  int                   `json:"totalCorrect"`
	Accuracy      decimal.Decimal       `json:"accuracy"`
	ByConfidence  QuizStatsByConfidence `json:"byConfidence"`
	Daily         []*QuizDailyScore     `json:"daily"`
}

// QuizStatsByConfidence 確信度（normal/strong）別の統計。
type QuizStatsByConfidence struct {
	Normal QuizConfidenceStats `json:"normal"`
	Strong QuizConfidenceStats `json:"strong"`
}

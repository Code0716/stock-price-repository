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

// QuizChart GET /quiz/chart のレスポンス（銘柄名・コードは含まない）。
type QuizChart struct {
	QuizDate string          `json:"quizDate"`
	Candles  []*ChartCandle  `json:"candles"`
	MA5      []*ChartMAPoint `json:"ma5"`
	MA25     []*ChartMAPoint `json:"ma25"`
	MA75     []*ChartMAPoint `json:"ma75"`
}

// QuizAnswerReveal 回答直後に公開する銘柄情報（POST /quiz/answers のレスポンス）。
type QuizAnswerReveal struct {
	TickerSymbol string `json:"tickerSymbol"`
	Name         string `json:"name"`
}

// QuizResultItem 採点済み1件の結果（銘柄名を公開）。
type QuizResultItem struct {
	QuestionOrder  int              `json:"questionOrder"`
	StockBrandID   string           `json:"stockBrandId"`
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

// QuizResults GET /quiz/results のレスポンス。QuizDate は出題基準日（quiz_date）。
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

// QuizDailyScore 日別スコア推移の1点。QuizDate は出題基準日（quiz_date）。
type QuizDailyScore struct {
	QuizDate string `json:"quizDate"`
	Answered int    `json:"answered"`
	Correct  int    `json:"correct"`
	Score    int    `json:"score"`
	// Pending 当該回答日のうち未採点（Outcome未確定）の件数。Answered/Score/Correctの分母には含まない。
	Pending int `json:"pending"`
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

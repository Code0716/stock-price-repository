//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"errors"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

// quizChartWarmupMonths MA75のウォームアップに十分な過去データを取得する月数。
const quizChartWarmupMonths = 9

// quizChartVisibleMonths チャートとして実際に表示する期間（月数）。
const quizChartVisibleMonths = 6

// ErrQuizQuestionNotFound 指定された quiz_date・銘柄の設問が存在しない。
var ErrQuizQuestionNotFound = errors.New("quiz question not found")

type quizInteractorImpl struct {
	quizDailyUniverseRepository          repositories.QuizDailyUniverseRepository
	quizAnswerRepository                 repositories.QuizAnswerRepository
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
	stockBrandRepository                 repositories.StockBrandRepository
}

type QuizInteractor interface {
	// GetQuestions 出題日の設問一覧と回答状況を返す（銘柄名は含まない）。dateがnilの場合は最新の出題日を使う。
	GetQuestions(ctx context.Context, date *time.Time) (*models.QuizQuestionSet, error)
	// GetChart 指定設問の匿名チャート（ローソク足+MA5/25/75+出来高）を返す。
	GetChart(ctx context.Context, quizDate time.Time, stockBrandID string) (*models.QuizChart, error)
	// SubmitAnswer 回答を1件登録する。既に回答済みの場合は repositories.ErrQuizAnswerAlreadyExists を返す。
	SubmitAnswer(ctx context.Context, quizDate time.Time, stockBrandID string, prediction models.QuizPrediction) error
	// GetResults 指定日の採点結果を返す（銘柄名を公開）。
	GetResults(ctx context.Context, quizDate time.Time) (*models.QuizResults, error)
	// GetStats 累計統計（スコア・的中率・確信度別的中率・日次推移）を返す。
	GetStats(ctx context.Context) (*models.QuizStats, error)
}

func NewQuizInteractor(
	quizDailyUniverseRepository repositories.QuizDailyUniverseRepository,
	quizAnswerRepository repositories.QuizAnswerRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	stockBrandRepository repositories.StockBrandRepository,
) QuizInteractor {
	return &quizInteractorImpl{
		quizDailyUniverseRepository:          quizDailyUniverseRepository,
		quizAnswerRepository:                 quizAnswerRepository,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		stockBrandRepository:                 stockBrandRepository,
	}
}

func (qi *quizInteractorImpl) resolveQuizDate(ctx context.Context, date *time.Time) (*time.Time, error) {
	if date != nil {
		return date, nil
	}
	return qi.quizDailyUniverseRepository.FindLatestQuizDate(ctx)
}

func (qi *quizInteractorImpl) GetQuestions(ctx context.Context, date *time.Time) (*models.QuizQuestionSet, error) {
	quizDate, err := qi.resolveQuizDate(ctx, date)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "resolveQuizDate error")
	}
	if quizDate == nil {
		return &models.QuizQuestionSet{Questions: []*models.QuizQuestion{}}, nil
	}

	universe, err := qi.quizDailyUniverseRepository.ListByQuizDate(ctx, *quizDate)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListByQuizDate error")
	}
	answers, err := qi.quizAnswerRepository.ListByQuizDate(ctx, *quizDate)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListByQuizDate(answers) error")
	}

	answerByBrand := make(map[string]*models.QuizAnswer, len(answers))
	for _, a := range answers {
		answerByBrand[a.StockBrandID] = a
	}

	questions := make([]*models.QuizQuestion, 0, len(universe))
	for _, u := range universe {
		q := &models.QuizQuestion{
			StockBrandID:  u.StockBrandID,
			QuestionOrder: u.QuestionOrder,
		}
		if a, ok := answerByBrand[u.StockBrandID]; ok {
			q.Answered = true
			prediction := string(a.Prediction)
			q.Prediction = &prediction
		}
		questions = append(questions, q)
	}

	return &models.QuizQuestionSet{
		QuizDate:      quizDate.Format(util.DateLayout),
		TotalCount:    len(universe),
		AnsweredCount: len(answers),
		Questions:     questions,
	}, nil
}

func (qi *quizInteractorImpl) GetChart(ctx context.Context, quizDate time.Time, stockBrandID string) (*models.QuizChart, error) {
	entry, err := qi.quizDailyUniverseRepository.FindByQuizDateAndStockBrandID(ctx, quizDate, stockBrandID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "FindByQuizDateAndStockBrandID error")
	}
	if entry == nil {
		return nil, ErrQuizQuestionNotFound
	}

	from := quizDate.AddDate(0, -quizChartWarmupMonths, 0)
	order := models.SortOrderAsc
	prices, err := qi.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: entry.TickerSymbol,
		DateFrom:     &from,
		DateTo:       &quizDate,
		DateOrder:    &order,
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	visibleFrom := quizDate.AddDate(0, -quizChartVisibleMonths, 0)
	return domain_service.BuildQuizChartSeries(prices, visibleFrom), nil
}

func (qi *quizInteractorImpl) SubmitAnswer(ctx context.Context, quizDate time.Time, stockBrandID string, prediction models.QuizPrediction) error {
	if !prediction.Valid() {
		return pkgerrors.New("invalid prediction")
	}

	entry, err := qi.quizDailyUniverseRepository.FindByQuizDateAndStockBrandID(ctx, quizDate, stockBrandID)
	if err != nil {
		return pkgerrors.Wrap(err, "FindByQuizDateAndStockBrandID error")
	}
	if entry == nil {
		return ErrQuizQuestionNotFound
	}

	answer := &models.QuizAnswer{
		QuizDate:     quizDate,
		StockBrandID: stockBrandID,
		TickerSymbol: entry.TickerSymbol,
		Prediction:   prediction,
		AnsweredAt:   time.Now(),
	}
	if err := qi.quizAnswerRepository.Create(ctx, answer); err != nil {
		if errors.Is(err, repositories.ErrQuizAnswerAlreadyExists) {
			return repositories.ErrQuizAnswerAlreadyExists
		}
		return pkgerrors.Wrap(err, "Create error")
	}
	return nil
}

func (qi *quizInteractorImpl) GetResults(ctx context.Context, quizDate time.Time) (*models.QuizResults, error) {
	universe, err := qi.quizDailyUniverseRepository.ListByQuizDate(ctx, quizDate)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListByQuizDate error")
	}
	answers, err := qi.quizAnswerRepository.ListByQuizDate(ctx, quizDate)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListByQuizDate(answers) error")
	}

	orderByBrand := make(map[string]int, len(universe))
	baseCloseByBrand := make(map[string]decimal.Decimal, len(universe))
	for _, u := range universe {
		orderByBrand[u.StockBrandID] = u.QuestionOrder
		baseCloseByBrand[u.StockBrandID] = u.BaseClosePrice
	}

	brandIDs := make([]string, 0, len(answers))
	for _, a := range answers {
		brandIDs = append(brandIDs, a.StockBrandID)
	}
	brands, err := qi.stockBrandRepository.FindByIDs(ctx, brandIDs)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "FindByIDs error")
	}
	nameByBrand := make(map[string]string, len(brands))
	for _, b := range brands {
		nameByBrand[b.ID] = b.Name
	}

	items := make([]*models.QuizResultItem, 0, len(answers))
	summary := models.QuizResultsSummary{}
	graded := len(answers) > 0

	for _, a := range answers {
		if !a.Graded() {
			graded = false
		}
		items = append(items, buildQuizResultItem(a, orderByBrand[a.StockBrandID], nameByBrand[a.StockBrandID], baseCloseByBrand[a.StockBrandID]))
		tallyQuizResultSummary(&summary, a)
	}

	return &models.QuizResults{
		QuizDate: quizDate.Format(util.DateLayout),
		Graded:   graded,
		Summary:  summary,
		Items:    items,
	}, nil
}

func buildQuizResultItem(a *models.QuizAnswer, questionOrder int, name string, baseClosePrice decimal.Decimal) *models.QuizResultItem {
	return &models.QuizResultItem{
		QuestionOrder:  questionOrder,
		TickerSymbol:   a.TickerSymbol,
		Name:           name,
		Prediction:     a.Prediction,
		BaseClosePrice: baseClosePrice,
		NextClosePrice: a.NextClosePrice,
		ActualReturn:   a.ActualReturn,
		Outcome:        a.Outcome,
		Score:          a.Score,
	}
}

func tallyQuizResultSummary(summary *models.QuizResultsSummary, a *models.QuizAnswer) {
	summary.Answered++
	if a.Score != nil {
		summary.Score += *a.Score
	}
	if a.Outcome == nil {
		return
	}
	switch *a.Outcome {
	case models.QuizOutcomeCorrect:
		summary.Correct++
	case models.QuizOutcomeIncorrect:
		summary.Incorrect++
	case models.QuizOutcomeDraw:
		summary.Draw++
	case models.QuizOutcomeVoid:
		summary.Void++
	}
}

func (qi *quizInteractorImpl) GetStats(ctx context.Context) (*models.QuizStats, error) {
	graded, err := qi.quizAnswerRepository.ListAllGraded(ctx)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListAllGraded error")
	}

	stats := &models.QuizStats{}
	totalDecided := 0
	normalAcc := &confidenceAccumulator{}
	strongAcc := &confidenceAccumulator{}

	type dailyAccumulator struct {
		answered, correct, score int
	}
	dailyByDate := make(map[string]*dailyAccumulator)
	var dailyOrder []string

	for _, a := range graded {
		if a.Score != nil {
			stats.TotalScore += *a.Score
		}
		stats.TotalAnswered++

		dateStr := a.QuizDate.Format(util.DateLayout)
		acc, ok := dailyByDate[dateStr]
		if !ok {
			acc = &dailyAccumulator{}
			dailyByDate[dateStr] = acc
			dailyOrder = append(dailyOrder, dateStr)
		}
		acc.answered++
		if a.Score != nil {
			acc.score += *a.Score
		}

		if a.Outcome == nil {
			continue
		}
		switch *a.Outcome {
		case models.QuizOutcomeCorrect:
			stats.TotalCorrect++
			acc.correct++
			if a.Prediction.IsStrong() {
				strongAcc.correct++
			} else {
				normalAcc.correct++
			}
			fallthrough
		case models.QuizOutcomeIncorrect:
			totalDecided++
			if a.Prediction.IsStrong() {
				strongAcc.answered++
			} else {
				normalAcc.answered++
			}
		}
	}

	stats.Accuracy = accuracyOf(stats.TotalCorrect, totalDecided)
	stats.ByConfidence = models.QuizStatsByConfidence{
		Normal: normalAcc.toStats(),
		Strong: strongAcc.toStats(),
	}

	stats.Daily = make([]*models.QuizDailyScore, 0, len(dailyOrder))
	for _, d := range dailyOrder {
		acc := dailyByDate[d]
		stats.Daily = append(stats.Daily, &models.QuizDailyScore{
			QuizDate: d,
			Answered: acc.answered,
			Correct:  acc.correct,
			Score:    acc.score,
		})
	}

	return stats, nil
}

type confidenceAccumulator struct {
	answered int
	correct  int
}

func (c *confidenceAccumulator) toStats() models.QuizConfidenceStats {
	return models.QuizConfidenceStats{
		Answered: c.answered,
		Correct:  c.correct,
		Accuracy: accuracyOf(c.correct, c.answered),
	}
}

// accuracyOf 正答率を計算する。分母（correct+incorrectの回答数）が0ならゼロを返す。
// draw/void は的中判定の対象外のため分母に含めない。
func accuracyOf(correct, answered int) decimal.Decimal {
	if answered == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(correct)).Div(decimal.NewFromInt(int64(answered)))
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"errors"
	"sort"
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

// quizChartRevealDays 答え合わせ用に出題基準日の翌営業日＋数日分を含める日数。
const quizChartRevealDays = 7

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
	// reveal=true の場合は答え合わせ用に出題基準日より後のローソク足も含める。
	GetChart(ctx context.Context, quizDate time.Time, stockBrandID string, reveal bool) (*models.QuizChart, error)
	// SubmitAnswer 回答を1件登録する。既に回答済みの場合は repositories.ErrQuizAnswerAlreadyExists を返す。
	// 登録成功時は回答直後に公開する銘柄情報（コード・名称）を返す。
	SubmitAnswer(ctx context.Context, quizDate time.Time, stockBrandID string, prediction models.QuizPrediction) (*models.QuizAnswerReveal, error)
	// GetResults 指定した出題基準日（quiz_date）の採点結果を返す（銘柄名を公開）。
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

func (qi *quizInteractorImpl) GetChart(ctx context.Context, quizDate time.Time, stockBrandID string, reveal bool) (*models.QuizChart, error) {
	entry, err := qi.quizDailyUniverseRepository.FindByQuizDateAndStockBrandID(ctx, quizDate, stockBrandID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "FindByQuizDateAndStockBrandID error")
	}
	if entry == nil {
		return nil, ErrQuizQuestionNotFound
	}

	from := quizDate.AddDate(0, -quizChartWarmupMonths, 0)
	dateTo := quizDate
	if reveal {
		// 答え合わせ用に出題基準日の翌営業日＋数日分のローソク足を含める。
		dateTo = quizDate.AddDate(0, 0, quizChartRevealDays)
	}
	order := models.SortOrderAsc
	prices, err := qi.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: entry.TickerSymbol,
		DateFrom:     &from,
		DateTo:       &dateTo,
		DateOrder:    &order,
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	visibleFrom := quizDate.AddDate(0, -quizChartVisibleMonths, 0)
	chart := domain_service.BuildQuizChartSeries(prices, visibleFrom)
	// BuildQuizChartSeries は最終足の日付を QuizDate に入れるため、front がマーカー位置に
	// 使う出題基準日を維持するようここで明示的に上書きする。
	chart.QuizDate = quizDate.Format(util.DateLayout)
	return chart, nil
}

func (qi *quizInteractorImpl) SubmitAnswer(ctx context.Context, quizDate time.Time, stockBrandID string, prediction models.QuizPrediction) (*models.QuizAnswerReveal, error) {
	if !prediction.Valid() {
		return nil, pkgerrors.New("invalid prediction")
	}

	entry, err := qi.quizDailyUniverseRepository.FindByQuizDateAndStockBrandID(ctx, quizDate, stockBrandID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "FindByQuizDateAndStockBrandID error")
	}
	if entry == nil {
		return nil, ErrQuizQuestionNotFound
	}

	name := ""
	brands, err := qi.stockBrandRepository.FindByIDs(ctx, []string{stockBrandID})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "FindByIDs error")
	}
	if len(brands) > 0 {
		name = brands[0].Name
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
			return nil, repositories.ErrQuizAnswerAlreadyExists
		}
		return nil, pkgerrors.Wrap(err, "Create error")
	}

	return &models.QuizAnswerReveal{
		TickerSymbol: entry.TickerSymbol,
		Name:         name,
	}, nil
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

	sort.Slice(answers, func(i, j int) bool {
		return orderByBrand[answers[i].StockBrandID] < orderByBrand[answers[j].StockBrandID]
	})

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
		StockBrandID:   a.StockBrandID,
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
	all, err := qi.quizAnswerRepository.ListAll(ctx)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ListAll error")
	}

	stats := &models.QuizStats{}
	totalDecided := 0
	normalAcc := &confidenceAccumulator{}
	strongAcc := &confidenceAccumulator{}

	type dailyAccumulator struct {
		answered, correct, score, pending int
	}
	dailyByDate := make(map[string]*dailyAccumulator)
	var dailyOrder []string

	for _, a := range all {
		// 日次集計キーは出題基準日（quiz_date）。
		dateStr := a.QuizDate.Format(util.DateLayout)
		acc, ok := dailyByDate[dateStr]
		if !ok {
			acc = &dailyAccumulator{}
			dailyByDate[dateStr] = acc
			dailyOrder = append(dailyOrder, dateStr)
		}
		acc.answered++
		stats.TotalAnswered++
		if a.Score != nil {
			stats.TotalScore += *a.Score
			acc.score += *a.Score
		}

		if !a.Graded() {
			acc.pending++
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

	// dailyOrder は ListAll の並び（answered_at 昇順）に由来するため、後から古いクイズに
	// 回答した場合などに quiz_date の昇順と一致しなくなる。ここで明示的に日付昇順へ並べ直す。
	sort.Strings(dailyOrder)

	stats.Daily = make([]*models.QuizDailyScore, 0, len(dailyOrder))
	for _, d := range dailyOrder {
		acc := dailyByDate[d]
		stats.Daily = append(stats.Daily, &models.QuizDailyScore{
			QuizDate: d,
			Answered: acc.answered,
			Correct:  acc.correct,
			Score:    acc.score,
			Pending:  acc.pending,
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

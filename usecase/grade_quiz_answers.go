//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type gradeQuizAnswersInteractorImpl struct {
	tx                                          repositories.Transaction
	quizAnswerRepository                        repositories.QuizAnswerRepository
	quizDailyUniverseRepository                 repositories.QuizDailyUniverseRepository
	stockBrandsDailyStockPriceRepository        repositories.StockBrandsDailyPriceRepository
	appliedStockSplitsHistoryRepository         repositories.AppliedStockSplitsHistoryRepository
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository
}

type GradeQuizAnswersInteractor interface {
	// GradeQuizAnswers 未採点の回答のうち、翌営業日の終値が確定しているものを採点する。
	GradeQuizAnswers(ctx context.Context) error
}

func NewGradeQuizAnswersInteractor(
	tx repositories.Transaction,
	quizAnswerRepository repositories.QuizAnswerRepository,
	quizDailyUniverseRepository repositories.QuizDailyUniverseRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	appliedStockSplitsHistoryRepository repositories.AppliedStockSplitsHistoryRepository,
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository,
) GradeQuizAnswersInteractor {
	return &gradeQuizAnswersInteractorImpl{
		tx:                                   tx,
		quizAnswerRepository:                 quizAnswerRepository,
		quizDailyUniverseRepository:          quizDailyUniverseRepository,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		appliedStockSplitsHistoryRepository:  appliedStockSplitsHistoryRepository,
		appliedStockConsolidationsHistoryRepository: appliedStockConsolidationsHistoryRepository,
	}
}

func (gi *gradeQuizAnswersInteractorImpl) GradeQuizAnswers(ctx context.Context) error {
	ungraded, err := gi.quizAnswerRepository.ListUngraded(ctx)
	if err != nil {
		return errors.Wrap(err, "ListUngraded error")
	}
	if len(ungraded) == 0 {
		return nil
	}

	byQuizDate := make(map[time.Time][]*models.QuizAnswer)
	for _, a := range ungraded {
		byQuizDate[a.QuizDate] = append(byQuizDate[a.QuizDate], a)
	}

	return gi.tx.DoInTx(ctx, func(ctx context.Context) error {
		for quizDate, answers := range byQuizDate {
			if err := gi.gradeQuizDate(ctx, quizDate, answers); err != nil {
				return errors.Wrapf(err, "gradeQuizDate error quizDate=%s", quizDate.Format("2006-01-02"))
			}
		}
		return nil
	})
}

func (gi *gradeQuizAnswersInteractorImpl) gradeQuizDate(ctx context.Context, quizDate time.Time, answers []*models.QuizAnswer) error {
	nextDate, err := gi.stockBrandsDailyStockPriceRepository.FindNextTradingDate(ctx, quizDate)
	if err != nil {
		return errors.Wrap(err, "FindNextTradingDate error")
	}
	if nextDate == nil {
		// 翌営業日がまだ来ていない（まだ引けていない）。次回バッチで再試行する。
		return nil
	}

	symbols := make([]string, 0, len(answers))
	seen := make(map[string]struct{})
	for _, a := range answers {
		if _, ok := seen[a.TickerSymbol]; ok {
			continue
		}
		seen[a.TickerSymbol] = struct{}{}
		symbols = append(symbols, a.TickerSymbol)
	}

	nextPrices, err := gi.stockBrandsDailyStockPriceRepository.ListRangePricesBySymbols(ctx, models.ListRangePricesBySymbolsFilter{
		Symbols:  symbols,
		DateFrom: nextDate,
		DateTo:   nextDate,
	})
	if err != nil {
		return errors.Wrap(err, "ListRangePricesBySymbols error")
	}
	nextCloseBySymbol := make(map[string]decimal.Decimal, len(nextPrices))
	for _, p := range nextPrices {
		nextCloseBySymbol[p.TickerSymbol] = p.Close
	}

	graded := make([]*models.QuizAnswer, 0, len(answers))
	for _, a := range answers {
		g, err := gi.gradeOneAnswer(ctx, quizDate, *nextDate, a, nextCloseBySymbol)
		if err != nil {
			return err
		}
		if g != nil {
			graded = append(graded, g)
		}
	}

	if err := gi.quizAnswerRepository.UpdateGrading(ctx, graded); err != nil {
		return errors.Wrap(err, "UpdateGrading error")
	}
	return nil
}

// gradeOneAnswer 1件の回答を採点する。設問（ユニバース）が見つからない場合は nil, nil を返す（対象外）。
func (gi *gradeQuizAnswersInteractorImpl) gradeOneAnswer(
	ctx context.Context,
	quizDate, nextDate time.Time,
	a *models.QuizAnswer,
	nextCloseBySymbol map[string]decimal.Decimal,
) (*models.QuizAnswer, error) {
	entry, err := gi.quizDailyUniverseRepository.FindByQuizDateAndStockBrandID(ctx, quizDate, a.StockBrandID)
	if err != nil {
		return nil, errors.Wrap(err, "FindByQuizDateAndStockBrandID error")
	}
	if entry == nil {
		return nil, nil
	}

	gradedAt := time.Now()
	a.GradedAt = &gradedAt

	nextClose, hasNextClose := nextCloseBySymbol[a.TickerSymbol]
	splitApplied, err := gi.appliedStockSplitsHistoryRepository.Exists(ctx, a.TickerSymbol, nextDate)
	if err != nil {
		return nil, errors.Wrap(err, "AppliedStockSplitsHistoryRepository.Exists error")
	}
	consolidationApplied, err := gi.appliedStockConsolidationsHistoryRepository.Exists(ctx, a.TickerSymbol, nextDate)
	if err != nil {
		return nil, errors.Wrap(err, "AppliedStockConsolidationsHistoryRepository.Exists error")
	}

	if !hasNextClose || splitApplied || consolidationApplied {
		// 売買停止等で翌営業日バーが無い、または株式分割・併合が発生した銘柄は公正に採点できないため void とする。
		voidOutcome := models.QuizOutcomeVoid
		zeroScore := 0
		a.Outcome = &voidOutcome
		a.Score = &zeroScore
		return a, nil
	}

	outcome, score, actualReturn := domain_service.GradeQuizAnswer(a.Prediction, entry.BaseClosePrice, nextClose)
	nc := nextClose
	a.NextClosePrice = &nc
	a.ActualReturn = &actualReturn
	a.Outcome = &outcome
	a.Score = &score
	return a, nil
}

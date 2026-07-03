package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func TestGradeQuizAnswersInteractorImpl_GradeQuizAnswers(t *testing.T) {
	quizDate := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	nextDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	t.Run("未採点が無ければ何もしない", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().ListUngraded(gomock.Any()).Return(nil, nil)

		// tx.DoInTx を含め、他のモックは呼ばれない
		interactor := NewGradeQuizAnswersInteractor(
			mock_repositories.NewMockTransaction(ctrl),
			answerRepo,
			mock_repositories.NewMockQuizDailyUniverseRepository(ctrl),
			mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
			mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl),
			mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl),
		)
		assert.NoError(t, interactor.GradeQuizAnswers(context.Background()))
	})

	t.Run("翌営業日がまだ来ていなければ採点をスキップする", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		answer := &models.QuizAnswer{ID: 1, QuizDate: quizDate, StockBrandID: "brand-a", TickerSymbol: "A001", Prediction: models.QuizPredictionUp}

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().ListUngraded(gomock.Any()).Return([]*models.QuizAnswer{answer}, nil)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().FindNextTradingDate(gomock.Any(), quizDate).Return(nil, nil)

		interactor := NewGradeQuizAnswersInteractor(
			tx,
			answerRepo,
			mock_repositories.NewMockQuizDailyUniverseRepository(ctrl),
			priceRepo,
			mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl),
			mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl),
		)
		assert.NoError(t, interactor.GradeQuizAnswers(context.Background()))
	})

	t.Run("正常系: 的中/翌日バー無しの両方を採点する", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		correctAnswer := &models.QuizAnswer{ID: 1, QuizDate: quizDate, StockBrandID: "brand-a", TickerSymbol: "A001", Prediction: models.QuizPredictionUp}
		voidAnswer := &models.QuizAnswer{ID: 2, QuizDate: quizDate, StockBrandID: "brand-b", TickerSymbol: "B001", Prediction: models.QuizPredictionDown}

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().ListUngraded(gomock.Any()).Return([]*models.QuizAnswer{correctAnswer, voidAnswer}, nil)
		answerRepo.EXPECT().UpdateGrading(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, answers []*models.QuizAnswer) error {
				assert.Len(t, answers, 2)
				byID := map[uint64]*models.QuizAnswer{}
				for _, a := range answers {
					byID[a.ID] = a
				}
				assert.Equal(t, models.QuizOutcomeCorrect, *byID[1].Outcome)
				assert.Equal(t, 1, *byID[1].Score)
				assert.Equal(t, models.QuizOutcomeVoid, *byID[2].Outcome)
				assert.Equal(t, 0, *byID[2].Score)
				return nil
			})

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().FindNextTradingDate(gomock.Any(), quizDate).Return(&nextDate, nil)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
			{TickerSymbol: "A001", Date: nextDate, Close: decimal.NewFromInt(110)},
			// B001は翌営業日バーが無い（売買停止想定）
		}, nil)

		universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-a").Return(
			&models.QuizUniverseEntry{StockBrandID: "brand-a", TickerSymbol: "A001", BaseClosePrice: decimal.NewFromInt(100)}, nil)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-b").Return(
			&models.QuizUniverseEntry{StockBrandID: "brand-b", TickerSymbol: "B001", BaseClosePrice: decimal.NewFromInt(200)}, nil)

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		splitRepo.EXPECT().Exists(gomock.Any(), "A001", nextDate).Return(false, nil)
		splitRepo.EXPECT().Exists(gomock.Any(), "B001", nextDate).Return(false, nil)

		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
		consolidationRepo.EXPECT().Exists(gomock.Any(), "A001", nextDate).Return(false, nil)
		consolidationRepo.EXPECT().Exists(gomock.Any(), "B001", nextDate).Return(false, nil)

		interactor := NewGradeQuizAnswersInteractor(tx, answerRepo, universeRepo, priceRepo, splitRepo, consolidationRepo)
		assert.NoError(t, interactor.GradeQuizAnswers(context.Background()))
	})

	t.Run("株式分割があった銘柄はvoidにする", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		answer := &models.QuizAnswer{ID: 3, QuizDate: quizDate, StockBrandID: "brand-c", TickerSymbol: "C001", Prediction: models.QuizPredictionUp}

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().ListUngraded(gomock.Any()).Return([]*models.QuizAnswer{answer}, nil)
		answerRepo.EXPECT().UpdateGrading(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, answers []*models.QuizAnswer) error {
				assert.Equal(t, models.QuizOutcomeVoid, *answers[0].Outcome)
				return nil
			})

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().FindNextTradingDate(gomock.Any(), quizDate).Return(&nextDate, nil)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
			{TickerSymbol: "C001", Date: nextDate, Close: decimal.NewFromInt(55)},
		}, nil)

		universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-c").Return(
			&models.QuizUniverseEntry{StockBrandID: "brand-c", TickerSymbol: "C001", BaseClosePrice: decimal.NewFromInt(100)}, nil)

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		splitRepo.EXPECT().Exists(gomock.Any(), "C001", nextDate).Return(true, nil)

		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
		consolidationRepo.EXPECT().Exists(gomock.Any(), "C001", nextDate).Return(false, nil)

		interactor := NewGradeQuizAnswersInteractor(tx, answerRepo, universeRepo, priceRepo, splitRepo, consolidationRepo)
		assert.NoError(t, interactor.GradeQuizAnswers(context.Background()))
	})
}

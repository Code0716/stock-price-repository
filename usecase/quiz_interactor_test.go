package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestQuizInteractorImpl_SubmitAnswer(t *testing.T) {
	quizDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	t.Run("不正な回答値はDBアクセスせずエラーになる", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		interactor := NewQuizInteractor(
			mock_repositories.NewMockQuizDailyUniverseRepository(ctrl),
			mock_repositories.NewMockQuizAnswerRepository(ctrl),
			mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
			mock_repositories.NewMockStockBrandRepository(ctrl),
		)
		_, err := interactor.SubmitAnswer(context.Background(), quizDate, "brand-a", models.QuizPrediction("invalid"))
		assert.Error(t, err)
	})

	t.Run("設問が存在しなければErrQuizQuestionNotFound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-a").Return(nil, nil)

		interactor := NewQuizInteractor(
			universeRepo,
			mock_repositories.NewMockQuizAnswerRepository(ctrl),
			mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
			mock_repositories.NewMockStockBrandRepository(ctrl),
		)
		_, err := interactor.SubmitAnswer(context.Background(), quizDate, "brand-a", models.QuizPredictionUp)
		assert.ErrorIs(t, err, ErrQuizQuestionNotFound)
	})

	t.Run("既に回答済みならErrQuizAnswerAlreadyExistsを伝播する", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-a").Return(
			&models.QuizUniverseEntry{StockBrandID: "brand-a", TickerSymbol: "A001"}, nil)

		stockBrandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		stockBrandRepo.EXPECT().FindByIDs(gomock.Any(), []string{"brand-a"}).Return(
			[]*models.StockBrand{{ID: "brand-a", Name: "銘柄A"}}, nil)

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(repositories.ErrQuizAnswerAlreadyExists)

		interactor := NewQuizInteractor(
			universeRepo,
			answerRepo,
			mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
			stockBrandRepo,
		)
		_, err := interactor.SubmitAnswer(context.Background(), quizDate, "brand-a", models.QuizPredictionUp)
		assert.True(t, errors.Is(err, repositories.ErrQuizAnswerAlreadyExists))
	})

	t.Run("正常系: 回答を作成する", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
		universeRepo.EXPECT().FindByQuizDateAndStockBrandID(gomock.Any(), quizDate, "brand-a").Return(
			&models.QuizUniverseEntry{StockBrandID: "brand-a", TickerSymbol: "A001"}, nil)

		stockBrandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
		stockBrandRepo.EXPECT().FindByIDs(gomock.Any(), []string{"brand-a"}).Return(
			[]*models.StockBrand{{ID: "brand-a", Name: "銘柄A"}}, nil)

		answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
		answerRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, a *models.QuizAnswer) error {
				assert.Equal(t, "A001", a.TickerSymbol)
				assert.Equal(t, models.QuizPredictionStrongUp, a.Prediction)
				return nil
			})

		interactor := NewQuizInteractor(
			universeRepo,
			answerRepo,
			mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
			stockBrandRepo,
		)
		reveal, err := interactor.SubmitAnswer(context.Background(), quizDate, "brand-a", models.QuizPredictionStrongUp)
		assert.NoError(t, err)
		assert.Equal(t, "A001", reveal.TickerSymbol)
		assert.Equal(t, "銘柄A", reveal.Name)
	})
}

func TestQuizInteractorImpl_GetQuestions(t *testing.T) {
	quizDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
	universeRepo.EXPECT().ListByQuizDate(gomock.Any(), quizDate).Return([]*models.QuizUniverseEntry{
		{StockBrandID: "brand-a", QuestionOrder: 1},
		{StockBrandID: "brand-b", QuestionOrder: 2},
	}, nil)

	answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
	answerRepo.EXPECT().ListByQuizDate(gomock.Any(), quizDate).Return([]*models.QuizAnswer{
		{StockBrandID: "brand-a", Prediction: models.QuizPredictionUp},
	}, nil)

	interactor := NewQuizInteractor(
		universeRepo,
		answerRepo,
		mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
		mock_repositories.NewMockStockBrandRepository(ctrl),
	)

	got, err := interactor.GetQuestions(context.Background(), &quizDate)
	assert.NoError(t, err)
	assert.Equal(t, 2, got.TotalCount)
	assert.Equal(t, 1, got.AnsweredCount)
	assert.True(t, got.Questions[0].Answered)
	assert.Equal(t, "up", *got.Questions[0].Prediction)
	assert.False(t, got.Questions[1].Answered)
	assert.Nil(t, got.Questions[1].Prediction)
}

func TestQuizInteractorImpl_GetResults_SortedByQuestionOrder(t *testing.T) {
	quizDate := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	universeRepo := mock_repositories.NewMockQuizDailyUniverseRepository(ctrl)
	universeRepo.EXPECT().ListByQuizDate(gomock.Any(), quizDate).Return([]*models.QuizUniverseEntry{
		{StockBrandID: "brand-a", QuestionOrder: 1, BaseClosePrice: decimal.NewFromInt(100)},
		{StockBrandID: "brand-b", QuestionOrder: 2, BaseClosePrice: decimal.NewFromInt(200)},
	}, nil)

	// リポジトリからは question_order と無関係な順で返ってくる想定
	answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
	answerRepo.EXPECT().ListByQuizDate(gomock.Any(), quizDate).Return([]*models.QuizAnswer{
		{StockBrandID: "brand-b", TickerSymbol: "B001", Prediction: models.QuizPredictionUp},
		{StockBrandID: "brand-a", TickerSymbol: "A001", Prediction: models.QuizPredictionDown},
	}, nil)

	stockBrandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	stockBrandRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return([]*models.StockBrand{
		{ID: "brand-a", Name: "銘柄A"},
		{ID: "brand-b", Name: "銘柄B"},
	}, nil)

	interactor := NewQuizInteractor(
		universeRepo,
		answerRepo,
		mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
		stockBrandRepo,
	)

	got, err := interactor.GetResults(context.Background(), quizDate)
	assert.NoError(t, err)
	assert.Len(t, got.Items, 2)
	assert.Equal(t, 1, got.Items[0].QuestionOrder)
	assert.Equal(t, "A001", got.Items[0].TickerSymbol)
	assert.Equal(t, 2, got.Items[1].QuestionOrder)
	assert.Equal(t, "B001", got.Items[1].TickerSymbol)
}

func TestQuizInteractorImpl_GetStats(t *testing.T) {
	quizDate1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	quizDate2 := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)

	correct := models.QuizOutcomeCorrect
	incorrect := models.QuizOutcomeIncorrect
	voidOutcome := models.QuizOutcomeVoid
	score1, score2, scoreMinus1, score0 := 1, 2, -1, 0

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	answerRepo := mock_repositories.NewMockQuizAnswerRepository(ctrl)
	answerRepo.EXPECT().ListAllGraded(gomock.Any()).Return([]*models.QuizAnswer{
		{QuizDate: quizDate1, Prediction: models.QuizPredictionUp, Outcome: &correct, Score: &score1},
		{QuizDate: quizDate1, Prediction: models.QuizPredictionStrongUp, Outcome: &correct, Score: &score2},
		{QuizDate: quizDate2, Prediction: models.QuizPredictionDown, Outcome: &incorrect, Score: &scoreMinus1},
		{QuizDate: quizDate2, Prediction: models.QuizPredictionUp, Outcome: &voidOutcome, Score: &score0},
	}, nil)

	interactor := NewQuizInteractor(
		mock_repositories.NewMockQuizDailyUniverseRepository(ctrl),
		answerRepo,
		mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl),
		mock_repositories.NewMockStockBrandRepository(ctrl),
	)

	stats, err := interactor.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, stats.TotalScore) // 1+2-1+0
	assert.Equal(t, 4, stats.TotalAnswered)
	assert.Equal(t, 2, stats.TotalCorrect)
	assert.True(t, stats.Accuracy.Equal(decimal.NewFromInt(2).Div(decimal.NewFromInt(3))), "voidを除いた correct+incorrect=3件中2件的中")

	assert.Equal(t, 2, stats.ByConfidence.Normal.Answered) // up(correct)とdown(incorrect)。voidは除外
	assert.Equal(t, 1, stats.ByConfidence.Normal.Correct)
	assert.Equal(t, 1, stats.ByConfidence.Strong.Answered)
	assert.Equal(t, 1, stats.ByConfidence.Strong.Correct)

	assert.Len(t, stats.Daily, 2)
	assert.Equal(t, "2026-07-01", stats.Daily[0].QuizDate)
	assert.Equal(t, 3, stats.Daily[0].Score) // 1+2
	assert.Equal(t, "2026-07-02", stats.Daily[1].QuizDate)
	assert.Equal(t, -1, stats.Daily[1].Score) // -1+0
}

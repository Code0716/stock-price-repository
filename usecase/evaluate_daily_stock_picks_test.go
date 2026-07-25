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

func evalTestPrice(symbol string, date time.Time, adjClose float64) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		TickerSymbol: symbol,
		Date:         date,
		Adjclose:     decimal.NewFromFloat(adjClose),
	}
}

func TestEvaluateDailyStockPicksInteractorImpl_EvaluateDailyStockPicks(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("pending 0件なら何もしない", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return(nil, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
		tx := mock_repositories.NewMockTransaction(ctrl)

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)
	})

	t.Run("5営業日後未到来なら部分更新でEvaluatedAtはnil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickDate := now.AddDate(0, 0, -2)
		pick := &models.DailyStockPick{PickDate: pickDate, StockBrandID: "b1", TickerSymbol: "1000"}

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return([]*models.DailyStockPick{pick}, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), models.ListRangePricesBySymbolsFilter{
			Symbols:  []string{"1000"},
			DateFrom: &pickDate,
			DateTo:   &now,
		}).Return([]*models.StockBrandDailyPrice{
			evalTestPrice("1000", pickDate, 100),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 1), 102),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 2), 101),
		}, nil)

		var updated []*models.DailyStockPick
		pickRepo.EXPECT().UpdateEvaluations(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				updated = picks
				return nil
			})

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)

		if assert.Len(t, updated, 1) {
			u := updated[0]
			assert.NotNil(t, u.Return1D)
			assert.Nil(t, u.Return3D)
			assert.Nil(t, u.Return5D)
			assert.Nil(t, u.Outcome)
			assert.Nil(t, u.EvaluatedAt)
		}
	})

	t.Run("5営業日後が到来していれば勝敗とEvaluatedAtが確定する", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickDate := now.AddDate(0, 0, -10)
		pick := &models.DailyStockPick{PickDate: pickDate, StockBrandID: "b1", TickerSymbol: "1000"}

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return([]*models.DailyStockPick{pick}, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
			evalTestPrice("1000", pickDate, 100),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 1), 101),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 2), 102),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 3), 103),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 4), 104),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 5), 110),
		}, nil)

		var updated []*models.DailyStockPick
		pickRepo.EXPECT().UpdateEvaluations(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				updated = picks
				return nil
			})

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		splitRepo.EXPECT().Exists(gomock.Any(), "1000", gomock.Any()).Return(false, nil).Times(5)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
		consolidationRepo.EXPECT().Exists(gomock.Any(), "1000", gomock.Any()).Return(false, nil).Times(5)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)

		if assert.Len(t, updated, 1) {
			u := updated[0]
			assert.NotNil(t, u.Return5D)
			assert.True(t, u.Return5D.Equal(decimal.RequireFromString("0.1")), "got %s", u.Return5D.String())
			assert.NotNil(t, u.Outcome)
			assert.Equal(t, models.DailyStockPickOutcomeWin, *u.Outcome)
			assert.NotNil(t, u.EvaluatedAt)
		}
	})

	t.Run("保有期間中に分割が適用されていたらvoid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickDate := now.AddDate(0, 0, -10)
		pick := &models.DailyStockPick{PickDate: pickDate, StockBrandID: "b1", TickerSymbol: "1000"}

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return([]*models.DailyStockPick{pick}, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return([]*models.StockBrandDailyPrice{
			evalTestPrice("1000", pickDate, 100),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 1), 101),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 2), 102),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 3), 103),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 4), 104),
			evalTestPrice("1000", pickDate.AddDate(0, 0, 5), 110),
		}, nil)

		var updated []*models.DailyStockPick
		pickRepo.EXPECT().UpdateEvaluations(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				updated = picks
				return nil
			})

		// 1回目のExistsチェックでtrueを返すため、以降の呼び出しは発生しない（短絡評価）。
		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		splitRepo.EXPECT().Exists(gomock.Any(), "1000", gomock.Any()).Return(true, nil).Times(1)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)

		if assert.Len(t, updated, 1) {
			u := updated[0]
			assert.NotNil(t, u.Outcome)
			assert.Equal(t, models.DailyStockPickOutcomeVoid, *u.Outcome)
			assert.NotNil(t, u.EvaluatedAt)
		}
	})

	t.Run("期限（30日）超過でバーが確定しない場合はvoid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickDate := now.AddDate(0, 0, -40)
		pick := &models.DailyStockPick{PickDate: pickDate, StockBrandID: "b1", TickerSymbol: "1000"}

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return([]*models.DailyStockPick{pick}, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		// 上場廃止等でバーが1本も無いケース
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(nil, nil)

		var updated []*models.DailyStockPick
		pickRepo.EXPECT().UpdateEvaluations(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				updated = picks
				return nil
			})

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)

		if assert.Len(t, updated, 1) {
			u := updated[0]
			assert.NotNil(t, u.Outcome)
			assert.Equal(t, models.DailyStockPickOutcomeVoid, *u.Outcome)
			assert.NotNil(t, u.EvaluatedAt)
		}
	})

	t.Run("複数pick_dateはそれぞれ別グループとして評価される", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pickDateA := now.AddDate(0, 0, -2)
		pickDateB := now.AddDate(0, 0, -3)
		pickA := &models.DailyStockPick{PickDate: pickDateA, StockBrandID: "b1", TickerSymbol: "1000"}
		pickB := &models.DailyStockPick{PickDate: pickDateB, StockBrandID: "b2", TickerSymbol: "2000"}

		pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
		pickRepo.EXPECT().ListPendingEvaluation(gomock.Any(), gomock.Any()).Return([]*models.DailyStockPick{pickA, pickB}, nil)

		priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		priceRepo.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, filter models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error) {
				assert.Len(t, filter.Symbols, 1)
				return []*models.StockBrandDailyPrice{
					evalTestPrice(filter.Symbols[0], *filter.DateFrom, 100),
				}, nil
			}).Times(2)

		var mu []string
		pickRepo.EXPECT().UpdateEvaluations(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				for _, p := range picks {
					mu = append(mu, p.TickerSymbol)
				}
				return nil
			}).Times(2)

		splitRepo := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
		consolidationRepo := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)

		tx := mock_repositories.NewMockTransaction(ctrl)
		tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

		interactor := NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
		err := interactor.EvaluateDailyStockPicks(context.Background(), now)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"1000", "2000"}, mu)
	})
}

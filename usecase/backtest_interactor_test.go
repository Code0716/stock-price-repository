package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/domain_service"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func genPrices(days int) []*models.StockBrandDailyPrice {
	base := time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, 0, days)
	for i := 0; i < days; i++ {
		c := 100 + float64(i%10) // 緩やかに変動
		out = append(out, &models.StockBrandDailyPrice{
			TickerSymbol: "7203",
			Date:         base.AddDate(0, 0, i),
			Open:         decimal.NewFromFloat(c),
			High:         decimal.NewFromFloat(c + 1),
			Low:          decimal.NewFromFloat(c - 1),
			Close:        decimal.NewFromFloat(c),
			Volume:       int64(100000 + i*1000),
		})
	}
	return out
}

func TestBacktestInteractor_GetBacktestComparison(t *testing.T) {
	from := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	order := models.SortOrderAsc
	wantFilter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: "7203",
		DateFrom:     &from,
		DateTo:       &to,
		DateOrder:    &order,
	}
	params := models.BacktestParams{
		TakeProfit:  decimal.NewFromFloat(0.10),
		StopLoss:    decimal.NewFromFloat(0.05),
		MaxHoldDays: 20,
	}

	t.Run("正常系: 全戦略をトータルリターン降順で返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		repo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return(genPrices(120), nil)

		interactor := NewBacktestInteractor(repo)
		got, err := interactor.GetBacktestComparison(context.Background(), "7203", &from, &to, params)
		assert.NoError(t, err)
		assert.Equal(t, "7203", got.Symbol)
		assert.Equal(t, 120, got.TradingDays)
		assert.Len(t, got.Strategies, len(domain_service.StrategyOrder))
		assert.Equal(t, "2021-01-04", got.From)
		// ランキング: 隣接でトータルリターンが降順
		for i := 1; i < len(got.Strategies); i++ {
			assert.True(t, got.Strategies[i-1].Result.TotalReturn.GreaterThanOrEqual(got.Strategies[i].Result.TotalReturn))
		}
		// 各戦略に表示名が付く
		for _, s := range got.Strategies {
			assert.NotEmpty(t, s.Label)
		}
	})

	t.Run("データ不足: 戦略は並ぶが取引0件", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		repo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return(genPrices(30), nil)

		interactor := NewBacktestInteractor(repo)
		got, err := interactor.GetBacktestComparison(context.Background(), "7203", &from, &to, params)
		assert.NoError(t, err)
		assert.Equal(t, 30, got.TradingDays)
		for _, s := range got.Strategies {
			assert.Equal(t, 0, s.Result.Trades)
		}
	})

	t.Run("異常系: 日足取得エラー", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
		repo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return(nil, errors.New("db error"))

		interactor := NewBacktestInteractor(repo)
		_, err := interactor.GetBacktestComparison(context.Background(), "7203", &from, &to, params)
		assert.Error(t, err)
	})
}

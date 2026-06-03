package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, client
}

func testBrands(symbols ...string) []*models.StockBrand {
	out := make([]*models.StockBrand, 0, len(symbols))
	for i, s := range symbols {
		out = append(out, &models.StockBrand{
			ID:           string(rune('A' + i)),
			TickerSymbol: s,
		})
	}
	return out
}

func testPrices(n int) []*models.StockBrandDailyPrice {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]*models.StockBrandDailyPrice, n)
	for i := 0; i < n; i++ {
		c := 100 + float64(i%10)
		prices[i] = &models.StockBrandDailyPrice{
			Date:   base.AddDate(0, 0, i),
			Open:   decimal.NewFromFloat(c),
			High:   decimal.NewFromFloat(c + 1),
			Low:    decimal.NewFromFloat(c - 1),
			Close:  decimal.NewFromFloat(c),
			Volume: int64(100000 + i*1000),
		}
	}
	return prices
}

func TestStrategyRankingInteractor_GetStrategyRanking_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, client := newTestRedis(t)

	interactor := NewStrategyRankingInteractor(nil, nil, client)
	got, err := interactor.GetStrategyRanking(context.Background())
	assert.NoError(t, err)
	assert.False(t, got.Computed)
	assert.Empty(t, got.Items)
}

func TestStrategyRankingInteractor_GetStrategyRanking_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr, client := newTestRedis(t)

	ranking := models.StrategyRanking{
		Computed:    true,
		ComputedAt:  "2026-06-01T00:00:00Z",
		Universe:    strategyRankingUniverse,
		TotalStocks: 10,
		Params: models.BacktestParams{
			TakeProfit:  decimal.NewFromFloat(0.10),
			StopLoss:    decimal.NewFromFloat(0.05),
			MaxHoldDays: 20,
		},
		Items: []models.StrategyRankingItem{
			{Strategy: "macd_bullish", Label: "MACD強気", StockCount: 10, AvgTotalReturn: decimal.NewFromFloat(0.05)},
		},
	}
	b, _ := json.Marshal(ranking)
	mr.Set(strategyRankingRedisKey, string(b))

	interactor := NewStrategyRankingInteractor(nil, nil, client)
	got, err := interactor.GetStrategyRanking(context.Background())
	assert.NoError(t, err)
	assert.True(t, got.Computed)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, "macd_bullish", got.Items[0].Strategy)
}

func TestStrategyRankingInteractor_ComputeAndSaveStrategyRanking_Normal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, client := newTestRedis(t)

	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)

	brands := testBrands("7203", "6758")
	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return(brands, nil)

	// 各銘柄について日足が返る（80日以上）
	prices := testPrices(90)
	asc := models.SortOrderAsc
	for range brands {
		priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error) {
				assert.Equal(t, asc, *filter.DateOrder)
				return prices, nil
			})
	}

	params := models.BacktestParams{
		TakeProfit:  decimal.NewFromFloat(0.10),
		StopLoss:    decimal.NewFromFloat(0.05),
		MaxHoldDays: 20,
	}

	interactor := NewStrategyRankingInteractor(brandRepo, priceRepo, client)
	n, err := interactor.ComputeAndSaveStrategyRanking(context.Background(), params, 5)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// Redisに保存されたことを確認
	got, err := interactor.GetStrategyRanking(context.Background())
	assert.NoError(t, err)
	assert.True(t, got.Computed)
	assert.Equal(t, 2, got.TotalStocks)
	assert.Len(t, got.Items, 5) // 5戦略
	// AvgTotalReturn 降順
	for i := 1; i < len(got.Items); i++ {
		assert.True(t, got.Items[i-1].AvgTotalReturn.GreaterThanOrEqual(got.Items[i].AvgTotalReturn))
	}
	// StockCount は検証した銘柄数
	for _, item := range got.Items {
		assert.Equal(t, 2, item.StockCount)
	}
}

func TestStrategyRankingInteractor_ComputeAndSaveStrategyRanking_SkipShortData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, client := newTestRedis(t)

	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)

	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return(testBrands("9999"), nil)
	// 79日分（minBacktestDays未満）→スキップ
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(testPrices(79), nil)

	params := models.BacktestParams{TakeProfit: decimal.NewFromFloat(0.1), StopLoss: decimal.NewFromFloat(0.05), MaxHoldDays: 20}
	interactor := NewStrategyRankingInteractor(brandRepo, priceRepo, client)
	n, err := interactor.ComputeAndSaveStrategyRanking(context.Background(), params, 5)
	assert.NoError(t, err)
	assert.Equal(t, 0, n) // スキップされたので処理0件

	// Redis には保存されているが StockCount=0
	got, err := interactor.GetStrategyRanking(context.Background())
	assert.NoError(t, err)
	assert.True(t, got.Computed)
	for _, item := range got.Items {
		assert.Equal(t, 0, item.StockCount)
	}
}

func TestStrategyRankingInteractor_ComputeAndSaveStrategyRanking_PriceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	_, client := newTestRedis(t)

	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)

	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return(testBrands("1234"), nil)
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

	params := models.BacktestParams{TakeProfit: decimal.NewFromFloat(0.1), StopLoss: decimal.NewFromFloat(0.05), MaxHoldDays: 20}
	interactor := NewStrategyRankingInteractor(brandRepo, priceRepo, client)
	_, err := interactor.ComputeAndSaveStrategyRanking(context.Background(), params, 5)
	assert.Error(t, err)
}

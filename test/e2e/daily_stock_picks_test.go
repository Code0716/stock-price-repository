package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

// dailyStockPickE2EBar 1本のOHLCVを組み立てる。
func dailyStockPickE2EBar(brandID, symbol string, date time.Time, close, high, low decimal.Decimal, volume int64) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		ID:           uuid.New().String(),
		StockBrandID: brandID,
		TickerSymbol: symbol,
		Date:         date,
		Open:         close,
		Close:        close,
		High:         high,
		Low:          low,
		Volume:       volume,
		Adjclose:     close,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// seedDailyStockPickWindow days本のうち最終日だけ大きく動く日足系列を作成する（最終日に MA 上抜けが点灯する設計）。
func seedDailyStockPickWindow(brandID, symbol string, base time.Time, days int, flatClose, lastClose decimal.Decimal) []*models.StockBrandDailyPrice {
	out := make([]*models.StockBrandDailyPrice, days)
	for i := 0; i < days-1; i++ {
		out[i] = dailyStockPickE2EBar(brandID, symbol, base.AddDate(0, 0, i), flatClose, flatClose.Add(decimal.NewFromInt(1)), flatClose.Sub(decimal.NewFromInt(1)), 2_000_000)
	}
	out[days-1] = dailyStockPickE2EBar(brandID, symbol, base.AddDate(0, 0, days-1), lastClose, lastClose.Add(decimal.NewFromInt(5)), lastClose.Sub(decimal.NewFromInt(5)), 6_000_000)
	return out
}

func TestE2E_DailyStockPicks(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	helper.TruncateAllTables(t, db)

	ctx := context.Background()
	stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
	priceRepo := database.NewStockBrandsDailyPriceRepositoryImpl(db)
	pickRepo := database.NewDailyStockPickRepositoryImpl(db)
	splitRepo := database.NewAppliedStockSplitsHistoryRepositoryImpl(db)
	consolidationRepo := database.NewAppliedStockConsolidationsHistoryRepositoryImpl(db)
	tx := database.NewTransaction(db)

	// 主要市場の銘柄を1件シード。120営業日フラット後、最終日に急騰して MA(5/25/75) を上抜ける。
	brandID := uuid.New().String()
	symbol := "1234"
	brand := &models.StockBrand{
		ID:           brandID,
		TickerSymbol: symbol,
		Name:         "テスト銘柄",
		MarketCode:   "111",
		MarketName:   "プライム",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	assert.NoError(t, stockBrandRepo.UpsertStockBrands(ctx, []*models.StockBrand{brand}))

	// pickDate は答え合わせの ListPendingEvaluation の取得窓（現在から60日以内）と
	// 期限（30日）の両方に収まるよう、現在から10日前にする。
	windowDays := 120
	pickDate := time.Now().AddDate(0, 0, -10)
	base := pickDate.AddDate(0, 0, -(windowDays - 1))
	prices := seedDailyStockPickWindow(brandID, symbol, base, windowDays, decimal.NewFromInt(1000), decimal.NewFromInt(2000))
	assert.NoError(t, priceRepo.CreateStockBrandDailyPrice(ctx, prices))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
	mockSlackAPI.EXPECT().
		SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("1234.5678", nil).
		AnyTimes()

	createInteractor := usecase.NewCreateDailyStockPicksInteractor(tx, priceRepo, stockBrandRepo, pickRepo, mockSlackAPI)
	createCmd := commands.NewCreateDailyStockPicksV1Command(createInteractor)
	evaluateInteractor := usecase.NewEvaluateDailyStockPicksInteractor(tx, pickRepo, priceRepo, splitRepo, consolidationRepo)
	evaluateCmd := commands.NewEvaluateDailyStockPicksV1Command(evaluateInteractor)

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		CreateDailyStockPicksV1Command:   createCmd,
		EvaluateDailyStockPicksV1Command: evaluateCmd,
		SlackAPIClient:                   mockSlackAPI,
	})

	t.Run("スクリーニングして保存・通知する", func(t *testing.T) {
		err := runner.Run(ctx, []string{"main", "create_daily_stock_picks_v1", "--top-n=3"})
		assert.NoError(t, err)

		var rows []*genModel.DailyStockPick
		assert.NoError(t, db.Where("stock_brand_id = ?", brandID).Find(&rows).Error)
		if assert.Len(t, rows, 1) {
			assert.Equal(t, uint32(1), rows[0].PickRank)
			assert.NotNil(t, rows[0].NotifiedAt)
			assert.Contains(t, rows[0].Strategies, "ma_cross")
		}
	})

	t.Run("再実行しても行が増えない（冪等）", func(t *testing.T) {
		err := runner.Run(ctx, []string{"main", "create_daily_stock_picks_v1", "--top-n=3"})
		assert.NoError(t, err)

		var count int64
		assert.NoError(t, db.Model(&genModel.DailyStockPick{}).Where("stock_brand_id = ?", brandID).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	})

	t.Run("翌営業日以降のバーが揃うと答え合わせが確定する", func(t *testing.T) {
		// pickDateの翌営業日から5営業日分のバーを追加投入する。
		followUp := make([]*models.StockBrandDailyPrice, 0, 5)
		closes := []decimal.Decimal{
			decimal.NewFromInt(2010), decimal.NewFromInt(2020), decimal.NewFromInt(2030),
			decimal.NewFromInt(2040), decimal.NewFromInt(2200),
		}
		for i, c := range closes {
			d := pickDate.AddDate(0, 0, i+1)
			followUp = append(followUp, dailyStockPickE2EBar(brandID, symbol, d, c, c.Add(decimal.NewFromInt(5)), c.Sub(decimal.NewFromInt(5)), 2_000_000))
		}
		assert.NoError(t, priceRepo.CreateStockBrandDailyPrice(ctx, followUp))

		err := runner.Run(ctx, []string{"main", "evaluate_daily_stock_picks_v1"})
		assert.NoError(t, err)

		var rows []*genModel.DailyStockPick
		assert.NoError(t, db.Where("stock_brand_id = ?", brandID).Find(&rows).Error)
		if assert.Len(t, rows, 1) {
			assert.NotNil(t, rows[0].Return5D)
			assert.NotNil(t, rows[0].Outcome)
			assert.Equal(t, "win", *rows[0].Outcome)
			assert.NotNil(t, rows[0].EvaluatedAt)
		}
	})
}

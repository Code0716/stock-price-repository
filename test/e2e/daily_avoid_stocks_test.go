package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

// avoidStockE2EBar 1本のOHLCVを組み立てる。
func avoidStockE2EBar(brandID, symbol string, date time.Time, close, high, low decimal.Decimal, volume int64) *models.StockBrandDailyPrice {
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

// seedAvoidStockWindow days本のうち最終日だけ大きく動く日足系列を作成する（ボラ計算に十分な本数を確保する）。
func seedAvoidStockWindow(brandID, symbol string, base time.Time, days int, flatClose, lastClose decimal.Decimal) []*models.StockBrandDailyPrice {
	out := make([]*models.StockBrandDailyPrice, days)
	for i := 0; i < days-1; i++ {
		out[i] = avoidStockE2EBar(brandID, symbol, base.AddDate(0, 0, i), flatClose, flatClose.Add(decimal.NewFromInt(1)), flatClose.Sub(decimal.NewFromInt(1)), 2_000_000)
	}
	out[days-1] = avoidStockE2EBar(brandID, symbol, base.AddDate(0, 0, days-1), lastClose, lastClose.Add(decimal.NewFromInt(5)), lastClose.Sub(decimal.NewFromInt(5)), 6_000_000)
	return out
}

func TestE2E_DailyAvoidStocks(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	helper.TruncateAllTables(t, db)

	ctx := context.Background()
	stockBrandRepo := database.NewStockBrandRepositoryImpl(db)
	priceRepo := database.NewStockBrandsDailyPriceRepositoryImpl(db)
	avoidRepo := database.NewDailyAvoidStockRepositoryImpl(db)
	tx := database.NewTransaction(db)

	// 主要市場の銘柄を1件シード。253営業日（ボラ計算に必要な252リターン分＋起点1本）フラット後、最終日に急騰する。
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

	windowDays := domain_service.DailyAvoidVolatilityWindowDays + 1 // 253
	asOfDate := time.Now().AddDate(0, 0, -1)
	base := asOfDate.AddDate(0, 0, -(windowDays - 1))
	prices := seedAvoidStockWindow(brandID, symbol, base, windowDays, decimal.NewFromInt(1000), decimal.NewFromInt(2000))
	assert.NoError(t, priceRepo.CreateStockBrandDailyPrice(ctx, prices))

	createInteractor := usecase.NewCreateDailyAvoidStocksInteractor(tx, priceRepo, stockBrandRepo, avoidRepo)
	createCmd := commands.NewCreateDailyAvoidStocksV1Command(createInteractor)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// 避けるべき銘柄バッチ自体はSlackへ送信しないが、runner共通の実行完了通知（dev_notification）用に必要。
	mockSlackAPI := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	mockSlackAPI.EXPECT().
		SendBlockMessage(gomock.Any(), gateway.SlackChannelNameDevNotification, gomock.Any()).
		Return(nil).
		AnyTimes()

	runner := helper.NewTestRunner(helper.TestRunnerOptions{
		CreateDailyAvoidStocksV1Command: createCmd,
		SlackAPIClient:                  mockSlackAPI,
	})

	t.Run("スクリーニングして保存する（通知はしない）", func(t *testing.T) {
		err := runner.Run(ctx, []string{"main", "create_daily_avoid_stocks_v1"})
		assert.NoError(t, err)

		var rows []*genModel.DailyAvoidStock
		assert.NoError(t, db.Where("stock_brand_id = ?", brandID).Find(&rows).Error)
		if assert.Len(t, rows, 1, "ユニバース1銘柄なら上位20%=1件として必ず該当する") {
			assert.Equal(t, uint32(1), rows[0].AvoidRank)
			assert.Equal(t, string(models.DailyAvoidSeverityHigh), rows[0].Severity)
			assert.Equal(t, models.DailyAvoidReasonHighVolatility, rows[0].Reason)
			assert.Equal(t, domain_service.DailyAvoidRuleVersion, rows[0].RuleVersion)
			assert.Equal(t, uint32(1), rows[0].UniverseSize)
		}

		var notificationCount int64
		assert.NoError(t, db.Table("notification_history").Count(&notificationCount).Error)
		assert.Equal(t, int64(0), notificationCount, "避けるべき銘柄バッチはnotification_historyに一切書かない")
	})

	t.Run("再実行しても行が増えない（冪等）", func(t *testing.T) {
		err := runner.Run(ctx, []string{"main", "create_daily_avoid_stocks_v1"})
		assert.NoError(t, err)

		var count int64
		assert.NoError(t, db.Model(&genModel.DailyAvoidStock{}).Where("stock_brand_id = ?", brandID).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	})

	t.Run("--forceで洗い替えても行数は変わらない", func(t *testing.T) {
		err := runner.Run(ctx, []string{"main", "create_daily_avoid_stocks_v1", "--force"})
		assert.NoError(t, err)

		var count int64
		assert.NoError(t, db.Model(&genModel.DailyAvoidStock{}).Where("stock_brand_id = ?", brandID).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	})
}

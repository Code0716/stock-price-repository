package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// rebuild_analyze_daily_prices
type RebuildAnalyzeDailyPricesV1Command struct {
	stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor
}

func NewRebuildAnalyzeDailyPricesV1Command(stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor) *RebuildAnalyzeDailyPricesV1Command {
	return &RebuildAnalyzeDailyPricesV1Command{stockBrandsDailyStockPriceInteractor}
}

func (c *RebuildAnalyzeDailyPricesV1Command) Command() *Command {
	return &Command{
		Name: "rebuild_analyze_daily_prices_v1",
		Usage: "stock_brands_daily_price_for_analyze をTRUNCATEし、過去5年分をJ-Quantsの調整済みOHLCVで作り直す。" +
			"破壊的操作のため --yes-truncate が必須。rawテーブルには触れない。",
		Action: c.Action,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "yes-truncate",
				Usage:    "stock_brands_daily_price_for_analyze / applied_stock_splits_history / applied_stock_consolidations_history をTRUNCATEすることに同意する",
				Required: true,
			},
		},
	}
}

func (c *RebuildAnalyzeDailyPricesV1Command) Action(ctx *cli.Context) error {
	if !ctx.Bool("yes-truncate") {
		return errors.New("--yes-truncate is required to run this destructive command")
	}

	err := c.stockBrandsDailyStockPriceInteractor.RebuildAnalyzeDailyPrices(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

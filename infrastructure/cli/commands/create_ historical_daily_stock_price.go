package commands

import (
	"time"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// create_historical_daily_stock_price
type CreateHistoricalDailyStockPricesV1Command struct {
	stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor
}

func NewCreateHistoricalDailyStockPricesV1Command(stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor) *CreateHistoricalDailyStockPricesV1Command {
	return &CreateHistoricalDailyStockPricesV1Command{stockBrandsDailyStockPriceInteractor}
}

func (c *CreateHistoricalDailyStockPricesV1Command) Command() *Command {
	return &Command{
		Name:   "create_historical_daily_stock_price",
		Usage:  "すべての銘柄の日足を取得する。",
		Action: c.Action,
	}
}

func (c *CreateHistoricalDailyStockPricesV1Command) Action(ctx *cli.Context) error {
	err := c.stockBrandsDailyStockPriceInteractor.CreateHistoricalDailyStockPrices(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

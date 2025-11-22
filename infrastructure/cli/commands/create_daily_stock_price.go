package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// create_daily_stock_price_v1
type CreateDailyStockPriceV1Command struct {
	stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor
}

func NewCreateDailyStockPriceV1Command(stockBrandsDailyStockPriceInteractor usecase.StockBrandsDailyPriceInteractor) *CreateDailyStockPriceV1Command {
	return &CreateDailyStockPriceV1Command{stockBrandsDailyStockPriceInteractor}
}

func (c *CreateDailyStockPriceV1Command) Command() *Command {
	return &Command{
		Name:   "create_daily_stock_price_v1",
		Usage:  "場終了後すべての銘柄の今日の日足を取得する。",
		Action: c.Action,
	}
}

func (c *CreateDailyStockPriceV1Command) Action(ctx *cli.Context) error {
	err := c.stockBrandsDailyStockPriceInteractor.CreateDailyStockPrice(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

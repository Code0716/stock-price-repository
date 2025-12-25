package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

type UpdateStockBrandsV1Command struct {
	stockBrandInteractor usecase.StockBrandInteractor
}

func NewUpdateStockBrandsV1Command(
	stockBrandInteractor usecase.StockBrandInteractor,
) *UpdateStockBrandsV1Command {
	return &UpdateStockBrandsV1Command{
		stockBrandInteractor,
	}
}

func (c *UpdateStockBrandsV1Command) Command() *Command {
	return &Command{
		Name:   "update_stock_brands_v1",
		Usage:  "株式銘柄をj-Quants取得してdbに保存する。",
		Action: c.Action,
	}
}

func (c *UpdateStockBrandsV1Command) Action(ctx *cli.Context) error {
	now := time.Now()
	err := c.stockBrandInteractor.UpdateStockBrands(ctx.Context, now)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

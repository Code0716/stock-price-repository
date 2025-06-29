package commands

import (
	"time"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// ExportStockBrandsAndDailyPriceToSQLV1Command
type ExportStockBrandsAndDailyPriceToSQLV1Command struct {
	exportSQLInteractor usecase.ExportSQLInteractor
}

func NewExportStockBrandsAndDailyPriceToSQLV1Command(
	exportSQLInteractor usecase.ExportSQLInteractor,
) *ExportStockBrandsAndDailyPriceToSQLV1Command {
	return &ExportStockBrandsAndDailyPriceToSQLV1Command{exportSQLInteractor}
}

func (c *ExportStockBrandsAndDailyPriceToSQLV1Command) Command() *Command {
	return &Command{
		Name:   "export_stock_brands_daily_stock_price_to_sql_v1",
		Usage:  "何某かの方法ででexportする",
		Action: c.Action,
	}
}

func (c *ExportStockBrandsAndDailyPriceToSQLV1Command) Action(ctx *cli.Context) error {
	now := time.Now()
	if err := c.exportSQLInteractor.ExportSQLFiles(ctx.Context, now); err != nil {
		return errors.Wrap(err, "failed to export SQL files")
	}

	return nil
}

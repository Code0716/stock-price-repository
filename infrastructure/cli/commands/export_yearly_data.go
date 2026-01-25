package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/urfave/cli/v2"
)

type ExportYearlyDataCommand struct {
	mySQLDumpClient gateway.MySQLDumpClient
}

func NewExportYearlyDataCommand(
	mySQLDumpClient gateway.MySQLDumpClient,
) *ExportYearlyDataCommand {
	return &ExportYearlyDataCommand{
		mySQLDumpClient: mySQLDumpClient,
	}
}

func (c *ExportYearlyDataCommand) Command() *Command {
	return &Command{
		Name:  "export_yearly_data",
		Usage: "Export yearly daily stock prices",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "year",
				Usage:    "Target year for export",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			year := ctx.Int("year")
			if year == 0 {
				year = time.Now().Year()
			}

			// 日足データ
			if err := c.mySQLDumpClient.ExportTableByYear(
				context.Background(),
				gateway.MySQLDumpTableNameStockBrandsDailyPrice,
				year,
			); err != nil {
				return fmt.Errorf("failed to export stock_brands_daily_stock_prices: %w", err)
			}

			// 分析用日足データ
			if err := c.mySQLDumpClient.ExportTableByYear(
				context.Background(),
				gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze,
				year,
			); err != nil {
				return fmt.Errorf("failed to export stock_brands_daily_stock_prices_for_analyze: %w", err)
			}

			fmt.Printf("Successfully exported yearly data for year %d\n", year)
			return nil
		},
	}
}

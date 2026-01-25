package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/urfave/cli/v2"
)

type ExportMasterDataCommand struct {
	mySQLDumpClient gateway.MySQLDumpClient
}

func NewExportMasterDataCommand(
	mySQLDumpClient gateway.MySQLDumpClient,
) *ExportMasterDataCommand {
	return &ExportMasterDataCommand{
		mySQLDumpClient: mySQLDumpClient,
	}
}

func (c *ExportMasterDataCommand) Command() *Command {
	return &Command{
		Name:  "export_master_data",
		Usage: "Export master data and other tables",
		Action: func(ctx *cli.Context) error {
			tables := []string{
				gateway.MySQLDumpTableNameStockBrand,
				gateway.MySQLDumpTableNameAppliedStockSplitsHistory,
				gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
				gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice,
			}

			now := time.Now()
			dateStr := now.Format("20060102")

			for _, table := range tables {
				fileName := fmt.Sprintf("%s_%s", dateStr, table)
				if err := c.mySQLDumpClient.ExportTableAll(
					context.Background(),
					fileName,
					table,
				); err != nil {
					return fmt.Errorf("failed to export table %s: %w", table, err)
				}
				fmt.Printf("Exported %s\n", table)
			}

			fmt.Println("Successfully exported master data")
			return nil
		},
	}
}

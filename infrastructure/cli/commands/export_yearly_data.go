package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/urfave/cli/v2"
)

type ExportYearlyDataCommand struct {
	mySQLDumpClient gateway.MySQLDumpClient
	boxClient       gateway.BoxClient
}

func NewExportYearlyDataCommand(
	mySQLDumpClient gateway.MySQLDumpClient,
	boxClient gateway.BoxClient,
) *ExportYearlyDataCommand {
	return &ExportYearlyDataCommand{
		mySQLDumpClient: mySQLDumpClient,
		boxClient:       boxClient,
	}
}

func (c *ExportYearlyDataCommand) Command() *Command {
	return &Command{
		Name:  "export_yearly_data",
		Usage: "Export yearly daily stock prices (all years if --year is omitted)",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "year",
				Usage: "Target year for export (省略時はDB内の全年を自動検出)",
			},
		},
		Action: func(ctx *cli.Context) error {
			tables := []string{
				gateway.MySQLDumpTableNameStockBrandsDailyPrice,
				gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze,
			}

			year := ctx.Int("year")
			var years []int
			if year != 0 {
				years = []int{year}
			} else {
				detected, err := c.mySQLDumpClient.GetDistinctYears(
					context.Background(),
					gateway.MySQLDumpTableNameStockBrandsDailyPrice,
				)
				if err != nil {
					return fmt.Errorf("failed to get distinct years: %w", err)
				}
				years = detected
			}

			for _, y := range years {
				for _, tableName := range tables {
					if err := c.mySQLDumpClient.ExportTableByYear(
						context.Background(),
						tableName,
						y,
					); err != nil {
						return fmt.Errorf("failed to export %s year=%d: %w", tableName, y, err)
					}

					filePath := filepath.Join(
						config.GetDatabase().ExportBackupPath,
						fmt.Sprintf("%d_%s.sql", y, tableName),
					)
					if err := c.boxClient.UploadFile(context.Background(), filePath); err != nil {
						log.Printf("WARN: box upload failed for %s: %v", filePath, err)
					} else if err := os.Remove(filePath); err != nil {
						log.Printf("WARN: failed to remove local file %s: %v", filePath, err)
					}
				}
				fmt.Printf("Successfully exported yearly data for year %d\n", y)
			}

			return nil
		},
	}
}

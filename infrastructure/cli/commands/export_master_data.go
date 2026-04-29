package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/urfave/cli/v2"
)

type ExportMasterDataCommand struct {
	mySQLDumpClient gateway.MySQLDumpClient
	boxClient       gateway.BoxClient
}

func NewExportMasterDataCommand(
	mySQLDumpClient gateway.MySQLDumpClient,
	boxClient gateway.BoxClient,
) *ExportMasterDataCommand {
	return &ExportMasterDataCommand{
		mySQLDumpClient: mySQLDumpClient,
		boxClient:       boxClient,
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

				filePath := filepath.Join(
					config.GetDatabase().ExportBackupPath,
					fileName+".sql",
				)
				if err := c.boxClient.UploadFile(context.Background(), filePath); err != nil {
					log.Printf("WARN: box upload failed for %s: %v", filePath, err)
				} else if err := os.Remove(filePath); err != nil {
					log.Printf("WARN: failed to remove local file %s: %v", filePath, err)
				}

				fmt.Printf("Exported %s\n", table)
			}

			fmt.Println("Successfully exported master data")
			return nil
		},
	}
}

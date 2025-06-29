//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/pkg/errors"
)

// mySQLDumpClientは、MySQLのダンプするためのクライアント。
type MySQLDumpClient struct {
	// redisClient *redis.Client
}

func NewMySQLDumpClient(
// redisClient *redis.Client,
) gateway.MySQLDumpClient {
	return &MySQLDumpClient{
		// redisClient,
	}
}

// ExportTableAllは、指定されたテーブルの全データをエクスポートする。
func (c *MySQLDumpClient) ExportTableAll(ctx context.Context, fileName, tableName string) error {
	dbConfig := config.Database()
	cmd := exec.Command("mysqldump",
		"-u"+dbConfig.User,
		"-p"+dbConfig.Passwd,
		"-h"+dbConfig.Host,
		"--skip-add-locks", // オプション：不要なLOCK文除外
		// "--no-create-info",  オプション：テーブルのCREATE文を除外
		dbConfig.DBName,
		tableName,
	)
	filePath := fmt.Sprintf("%s/%s.sql", dbConfig.ExportBackupPath, fileName)

	outFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "os.Create error")
	}
	defer outFile.Close()
	cmd.Stdout = outFile

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "mysqldump command run error")
	}

	log.Printf("mysqldump & export success: %s\n", fileName)
	return nil
}

// ExportDailyStockPriceByYear 各銘柄の日足を年ごとにexportする
func (c *MySQLDumpClient) ExportDailyStockPriceByYear(ctx context.Context, yearFrom, yearTo int) error {
	if yearFrom > yearTo {
		return errors.New("yearFrom must be less than or equal to yearTo")
	}

	for i := yearFrom; i <= yearTo; i++ {
		if err := c.exportDailyStockPriceByYear(ctx, i); err != nil {
			return errors.Wrapf(err, "exportYearlyData error for year %d", i)
		}
	}

	return nil
}

// exportDailyStockPriceByYearは、指定された年の日足データをエクスポートする。
func (c *MySQLDumpClient) exportDailyStockPriceByYear(ctx context.Context, year int) error {
	dbConfig := config.Database()
	where := fmt.Sprintf("YEAR(date) = %d", year)

	tableName := gateway.MySQLDumpTableNameStockBrandsDailyPrice
	cmd := exec.Command("mysqldump",
		"-u"+dbConfig.User,
		"-p"+dbConfig.Passwd,
		"-h"+dbConfig.Host,
		"--skip-add-locks", // オプション：不要なLOCK文除外
		"--no-create-info", // オプション：テーブルのCREATE文を除外
		"--where="+where,
		dbConfig.DBName,
		tableName, // テーブル名を指定
	)

	filePath := fmt.Sprintf("%s/%d_%s.sql", dbConfig.ExportBackupPath, year, tableName)
	outFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "os.Create error")
	}
	defer outFile.Close()
	cmd.Stdout = outFile

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "mysqldump command run error")
	}

	log.Printf("mysqldump & export success for year %d: %s\n", year, tableName)
	return nil
}

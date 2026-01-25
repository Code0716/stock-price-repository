//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

var execCommand = exec.Command

// mySQLDumpClientは、MySQLのダンプするためのクライアント。
type MySQLDumpClient struct {
	// redisClient *redis.Client
}

func NewMySQLDumpClient() gateway.MySQLDumpClient {
	return &MySQLDumpClient{}
}

// ExportTableAllは、指定されたテーブルの全データをエクスポートする。
func (c *MySQLDumpClient) ExportTableAll(_ context.Context, fileName, tableName string) error {
	dbConfig := config.GetDatabase()
	cmd := execCommand("mysqldump",
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

// ExportTableByYear 指定したテーブルを年ごとにexportする
func (c *MySQLDumpClient) ExportTableByYear(_ context.Context, tableName string, year int) error {
	if year < 0 {
		return errors.New("year must be a positive integer")
	}

	if err := c.exportTableByYear(tableName, year); err != nil {
		return errors.Wrapf(err, "exportYearlyData error for year %d", year)
	}

	return nil
}

// exportTableByYearは、指定された年の日足データをエクスポートする。
func (c *MySQLDumpClient) exportTableByYear(tableName string, year int) error {
	dbConfig := config.GetDatabase()
	where := fmt.Sprintf("YEAR(date) = %d", year)

	cmd := execCommand("mysqldump",
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

	return nil
}

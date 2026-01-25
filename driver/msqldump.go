//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

var execCommandContext = exec.CommandContext

// mySQLDumpClientは、MySQLのダンプするためのクライアント。
type MySQLDumpClient struct {
	// redisClient *redis.Client
}

func NewMySQLDumpClient() gateway.MySQLDumpClient {
	return &MySQLDumpClient{}
}

// ExportTableAllは、指定されたテーブルの全データをエクスポートする。
func (c *MySQLDumpClient) ExportTableAll(ctx context.Context, fileName, tableName string) error {
	dbConfig := config.GetDatabase()

	if err := os.MkdirAll(dbConfig.ExportBackupPath, 0755); err != nil {
		return errors.Wrap(err, "failed to create backup directory")
	}

	cmd := execCommandContext(ctx, "mysqldump",
		"-u"+dbConfig.User,
		"-h"+dbConfig.Host,
		"--skip-add-locks", // オプション：不要なLOCK文除外
		// "--no-create-info",  オプション：テーブルのCREATE文を除外
		dbConfig.DBName,
		tableName,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", dbConfig.Passwd))

	filePath := filepath.Join(dbConfig.ExportBackupPath, fmt.Sprintf("%s.sql", fileName))

	outFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "os.Create error")
	}
	defer outFile.Close()
	cmd.Stdout = outFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "mysqldump command run error: %s", stderr.String())
	}

	log.Printf("mysqldump & export success: %s\n", fileName)
	return nil
}

// ExportTableByYear 指定したテーブルを年ごとにexportする
func (c *MySQLDumpClient) ExportTableByYear(ctx context.Context, tableName string, year int) error {
	if year < 0 {
		return errors.New("year must be a positive integer")
	}

	if err := c.exportTableByYear(ctx, tableName, year); err != nil {
		return errors.Wrapf(err, "exportYearlyData error for year %d", year)
	}

	return nil
}

// exportTableByYearは、指定された年の日足データをエクスポートする。
func (c *MySQLDumpClient) exportTableByYear(ctx context.Context, tableName string, year int) error {
	dbConfig := config.GetDatabase()
	where := fmt.Sprintf("YEAR(date) = %d", year)

	if err := os.MkdirAll(dbConfig.ExportBackupPath, 0755); err != nil {
		return errors.Wrap(err, "failed to create backup directory")
	}

	cmd := execCommandContext(ctx, "mysqldump",
		"-u"+dbConfig.User,
		"-h"+dbConfig.Host,
		"--skip-add-locks", // オプション：不要なLOCK文除外
		"--no-create-info", // オプション：テーブルのCREATE文を除外
		"--where="+where,
		dbConfig.DBName,
		tableName, // テーブル名を指定
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", dbConfig.Passwd))

	filePath := filepath.Join(dbConfig.ExportBackupPath, fmt.Sprintf("%d_%s.sql", year, tableName))
	outFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "os.Create error")
	}
	defer outFile.Close()
	cmd.Stdout = outFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "mysqldump command run error: %s", stderr.String())
	}

	return nil
}
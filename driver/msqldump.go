//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
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
		"--protocol=tcp",
		"-u"+dbConfig.User,
		"-h"+dbConfig.Host,
		"-P"+dbConfig.Port,
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

// GetDistinctYears 指定したテーブルに存在する年の一覧を昇順で返す
func (c *MySQLDumpClient) GetDistinctYears(ctx context.Context, tableName string) ([]int, error) {
	dbConfig := config.GetDatabase()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbConfig.User, dbConfig.Passwd, dbConfig.Host, dbConfig.Port, dbConfig.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "sql.Open error")
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT DISTINCT YEAR(date) FROM `%s` ORDER BY YEAR(date) ASC", tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "db.QueryContext error")
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, errors.Wrap(err, "rows.Scan error")
		}
		years = append(years, year)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows.Err")
	}

	return years, nil
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
		"--protocol=tcp",
		"-u"+dbConfig.User,
		"-h"+dbConfig.Host,
		"-P"+dbConfig.Port,
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

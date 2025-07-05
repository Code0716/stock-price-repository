package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/pkg/errors"
)

// ExportSQLFiles mysqlをsqlファイルにバックアップする。
func (ei *exportSQLInteractorImpl) ExportSQLFiles(ctx context.Context, now time.Time) error {

	tableNames := []string{
		gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
		gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice,
		gateway.MySQLDumpTableNameStockBrand,
		gateway.MySQLDumpTableNameSector33AverageDailyPrice,
	}
	for _, tableName := range tableNames {
		if err := ei.exportTableAll(ctx, tableName, now); err != nil {
			return errors.Wrapf(err, "exportTableAll %s error", tableName)
		}
	}

	errCh := make(chan error, len(tableNames))
	for _, tableName := range tableNames {
		tName := tableName // goroutine内で正しくキャプチャするため
		go func() {
			if err := ei.exportTableAll(ctx, tName, now); err != nil {
				errCh <- errors.Wrapf(err, "exportTableAll %s error", tName)
				return
			}
			errCh <- nil
		}()
	}

	for range tableNames {
		if err := <-errCh; err != nil {
			log.Printf("Error occurred: %v", err)
		}
	}

	// Export daily stock prices
	// ゴルーチンでやると、重い処理を同時にやることになるので、DB側に負荷がかかる。
	// レコード数も多いことから順次実行でいいと思う。
	for year := 2019; year <= now.Year(); year++ {
		if err := ei.mySQLDumpClient.ExportDailyStockPriceByYear(
			ctx,
			year,
		); err != nil {
			return errors.Wrap(err, "ExportDailyStockPriceByYear error")
		}
	}
	return nil
}

// exportTableAll 各テーブルを出力する
func (ei *exportSQLInteractorImpl) exportTableAll(ctx context.Context, tableName string, now time.Time) error {
	// 各銘柄日足を年ごとにexport
	if err := ei.mySQLDumpClient.ExportTableAll(ctx,
		fmt.Sprintf("%s_%s",
			util.DatetimeToFileNameDateStr(now),
			tableName,
		),
		tableName,
	); err != nil {
		return errors.Wrap(err, "ExportTableAll StockBrand error")
	}
	return nil
}

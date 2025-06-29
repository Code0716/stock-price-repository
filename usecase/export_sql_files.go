package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/pkg/errors"
)

// ExportSQLFiles mysqlをsqlファイルにバックアップする。
func (ei *exportSQLInteractorImpl) ExportSQLFiles(ctx context.Context, now time.Time) error {

	tableNames := []string{
		gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
		gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
		gateway.MySQLDumpTableNameStockBrand,
	}
	for _, tableName := range tableNames {
		if err := ei.exportTableAll(ctx, tableName, now); err != nil {
			return errors.Wrapf(err, "exportTableAll %s error", tableName)
		}
	}

	// Export daily stock prices
	if err := ei.mySQLDumpClient.ExportDailyStockPriceByYear(
		ctx,
		now.AddDate(-5, 0, 0).Year(),
		now.Year(),
	); err != nil {
		return errors.Wrap(err, "ExportDailyStockPriceByYear error")
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
		gateway.MySQLDumpTableNameStockBrand,
	); err != nil {
		return errors.Wrap(err, "ExportTableAll StockBrand error")
	}
	return nil
}

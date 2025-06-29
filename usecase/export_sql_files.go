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

	if err := ei.mySQLDumpClient.ExportTableAll(ctx,
		fmt.Sprintf("%s_%s",
			util.DatetimeToFileNameDateStr(now),
			gateway.MySQLDumpTableNameStockBrand,
		),
		gateway.MySQLDumpTableNameStockBrand,
	); err != nil {
		return errors.Wrap(err, "ExportTableAll StockBrand error")
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

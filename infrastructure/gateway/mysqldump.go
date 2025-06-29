//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package gateway

import (
	"context"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
)

// テーブル名を定義
const (
	MySQLDumpTableNameStockBrand                     = genModel.TableNameStockBrand
	MySQLDumpTableNameStockBrandsDailyPrice          = genModel.TableNameStockBrandsDailyPrice
	MySQLDumpTableNameNikkeiStockAverageDailyPrice   = genModel.TableNameNikkeiStockAverageDailyPrice
	MySQLDumpTableNameDjiStockAverageDailyStockPrice = genModel.TableNameDjiStockAverageDailyStockPrice
	MySQLDumpTableNameSector33AverageDailyPrice      = genModel.TableNameSector33AverageDailyPrice
	MySQLDumpTableNameSector17AverageDailyPrice      = genModel.TableNameSector17AverageDailyPrice
)

type MySQLDumpClient interface {
	// 指定したテーブルを全件exportする.
	ExportTableAll(ctx context.Context, fileName, tableName string) error
	// 各銘柄の日足を年ごとにexportす
	ExportDailyStockPriceByYear(ctx context.Context, yearFrom, yearTo int) error
}

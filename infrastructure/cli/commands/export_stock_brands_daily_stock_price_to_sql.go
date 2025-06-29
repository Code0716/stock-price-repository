package commands

import (
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/urfave/cli/v2"
)

// export_stock_brands_daily_stock_price_to_sql
type ExportStockBrandsDailyPriceToSQLV1Command struct {
}

func NewExportStockBrandsDailyPriceToSQLV1Command(stockBrandInteractor usecase.StockBrandInteractor) *ExportStockBrandsDailyPriceToSQLV1Command {
	return &ExportStockBrandsDailyPriceToSQLV1Command{}
}

func (c *ExportStockBrandsDailyPriceToSQLV1Command) Command() *Command {
	return &Command{
		Name:   "export_stock_brands_daily_stock_price_to_sql_v1",
		Usage:  "何某かの方法ででexportする",
		Action: c.Action,
	}
}

func (c *ExportStockBrandsDailyPriceToSQLV1Command) Action(ctx *cli.Context) error {
	// TODO 実装
	// now := time.Now()
	// dbConfig := config.Database()

	// exportPath := filepath.Join(dbConfig.ExportPath, fmt.Sprintf("/stock_price_repository_%s.sql", strings.Replace(util.DatetimeToDateStr(now), "-", "_", 2)))
	// fmt.Printf("ここやで%v \n", exportPath)
	// cmd := exec.Command("mysqldump", "-u", dbConfig.User, "-p", dbConfig.Passwd, dbConfig.DBName)

	// out, err := exec.Command("sh", "-c", fmt.Sprintf("%s > %s", cmd.String(), exportPath)).CombinedOutput()
	// if err != nil {
	// 	log.Printf("出力: %s", string(out))
	// 	return errors.Wrap(err, "exec.Command error")
	// }

	return nil
}

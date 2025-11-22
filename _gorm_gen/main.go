package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/driver"
)

func main() {
	config.LoadEnvConfig()

	dsn, err := driver.BuildMySQLConnectionString()
	if err != nil {
		panic(err)
	}

	g := gen.NewGenerator(
		gen.Config{
			OutPath:          "./infrastructure/database/gen_query",
			Mode:             gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
			ModelPkgPath:     "./infrastructure/database/gen_model",
			FieldNullable:    true,
			FieldSignable:    true,
			FieldWithTypeTag: true,
		},
	)

	// gormでDBに接続
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(err)
	}
	g.UseDB(db)
	g.ApplyBasic(g.GenerateAllTable()...)
	// g.GenerateModel("stock_brand")
	// g.GenerateModel("stock_brands_daily_price")
	// g.GenerateModel("nikkei_stock_average_daily_price")
	// g.GenerateModel("dji_stock_average_daily_stock_price")
	// g.ApplyInterface(func(Repository) {}, g.GenerateAllTable()...)
	g.Execute()

}

//go:build wireinject

package di

import (
	"context"
	"net/http"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	"github.com/Code0716/stock-price-repository/entrypoint/api/router"
	"github.com/Code0716/stock-price-repository/infrastructure/cli"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/usecase"

	"github.com/google/wire"
)

var usecaseSet = wire.NewSet(
	usecase.NewStockBrandInteractor,
	usecase.NewIndexInteractor,
	usecase.NewStockBrandsDailyPriceInteractor,
	usecase.NewExportSQLInteractor,
)

var driverSet = wire.NewSet(
	driver.NewGorm,
	driver.NewDBConn,
	driver.NewHTTPRequest,
	driver.NewHTTPServer,
	driver.NewSlackAPIClient,
	driver.OpenRedis,
	driver.NewStockAPIClient,
	driver.NewMySQLDumpClient,
	driver.NewLogger,
)

var cliSet = wire.NewSet(
	cli.NewRunner,
	commands.NewHealthCheckCommand,
	commands.NewSetJQuantsAPITokenToRedisV1Command,
	commands.NewUpdateStockBrandsV1Command,
	commands.NewCreateHistoricalDailyStockPricesV1Command,
	commands.NewCreateDailyStockPriceV1Command,
	commands.NewExportStockBrandsAndDailyPriceToSQLV1Command,
	commands.NewCreateNikkeiAndDjiHistoricalDataV1Command,
)

var databaseSet = wire.NewSet(
	database.NewTransaction,
	database.NewStockBrandRepositoryImpl,
	database.NewNikkeiRepositoryImpl,
	database.NewDjiRepositoryImpl,
	database.NewStockBrandsDailyPriceRepositoryImpl,
	database.NewAnalyzeStockBrandPriceHistoryRepositoryImpl,
	database.NewStockBrandsDailyPriceForAnalyzeRepositoryImpl,
)

func InitializeCli(ctx context.Context) (*cli.Runner, func(), error) {
	wire.Build(
		cliSet,
		usecaseSet,
		databaseSet,
		driverSet,
	)
	return nil, nil, nil
}

var apiSet = wire.NewSet(
	handler.NewStockPriceHandler,
	handler.NewStockBrandHandler,
	router.NewRouter,
)

func InitializeApiServer(ctx context.Context) (*http.ServeMux, func(), error) {
	wire.Build(
		apiSet,
		usecaseSet,
		databaseSet,
		driverSet,
	)
	return nil, nil, nil
}

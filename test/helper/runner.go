package helper

import (
	"github.com/Code0716/stock-price-repository/infrastructure/cli"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/usecase"
)

type TestRunnerOptions struct {
	HealthCheckCommand                        *commands.HealthCheckCommand
	UpdateStockBrandsV1Command                *commands.UpdateStockBrandsV1Command
	CreateHistoricalDailyStockPricesV1Command *commands.CreateHistoricalDailyStockPricesV1Command
	CreateDailyStockPriceV1Command            *commands.CreateDailyStockPriceV1Command
	CreateNikkeiAndDjiHistoricalDataV1Command *commands.CreateNikkeiAndDjiHistoricalDataV1Command
	AdjustHistoricalDataForStockSplitCommand  *commands.AdjustHistoricalDataForStockSplitCommand
	ExportYearlyDataCommand                   *commands.ExportYearlyDataCommand
	ExportMasterDataCommand                   *commands.ExportMasterDataCommand
	IndexInteractor                           usecase.IndexInteractor
	SlackAPIClient                            gateway.SlackAPIClient
	MySQLDumpClient                           gateway.MySQLDumpClient
	BoxClient                                 gateway.BoxClient
}

func NewTestRunner(opts TestRunnerOptions) *cli.Runner {
	if opts.HealthCheckCommand == nil {
		opts.HealthCheckCommand = commands.NewHealthCheckCommand(nil)
	}
	if opts.UpdateStockBrandsV1Command == nil {
		opts.UpdateStockBrandsV1Command = commands.NewUpdateStockBrandsV1Command(nil)
	}
	if opts.CreateHistoricalDailyStockPricesV1Command == nil {
		opts.CreateHistoricalDailyStockPricesV1Command = commands.NewCreateHistoricalDailyStockPricesV1Command(nil)
	}
	if opts.CreateDailyStockPriceV1Command == nil {
		opts.CreateDailyStockPriceV1Command = commands.NewCreateDailyStockPriceV1Command(nil)
	}
	if opts.CreateNikkeiAndDjiHistoricalDataV1Command == nil {
		opts.CreateNikkeiAndDjiHistoricalDataV1Command = commands.NewCreateNikkeiAndDjiHistoricalDataV1Command(nil)
	}
	if opts.AdjustHistoricalDataForStockSplitCommand == nil {
		opts.AdjustHistoricalDataForStockSplitCommand = commands.NewAdjustHistoricalDataForStockSplitCommand(nil)
	}
	if opts.ExportYearlyDataCommand == nil {
		opts.ExportYearlyDataCommand = commands.NewExportYearlyDataCommand(opts.MySQLDumpClient, opts.BoxClient)
	}
	if opts.ExportMasterDataCommand == nil {
		opts.ExportMasterDataCommand = commands.NewExportMasterDataCommand(opts.MySQLDumpClient, opts.BoxClient)
	}

	return cli.NewRunner(
		opts.HealthCheckCommand,
		opts.UpdateStockBrandsV1Command,
		opts.CreateHistoricalDailyStockPricesV1Command,
		opts.CreateDailyStockPriceV1Command,
		opts.CreateNikkeiAndDjiHistoricalDataV1Command,
		opts.AdjustHistoricalDataForStockSplitCommand,
		opts.ExportYearlyDataCommand,
		opts.ExportMasterDataCommand,
		opts.IndexInteractor,
		opts.SlackAPIClient,
	)
}

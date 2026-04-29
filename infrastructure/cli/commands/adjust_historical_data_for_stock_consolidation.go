package commands

import (
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

// AdjustHistoricalDataForStockConsolidationCommand is a command to adjust historical data for stock consolidations (reverse splits).
type AdjustHistoricalDataForStockConsolidationCommand struct {
	adjustHistoricalDataForStockConsolidationInteractor usecase.AdjustHistoricalDataForStockConsolidation
}

// NewAdjustHistoricalDataForStockConsolidationCommand creates a new AdjustHistoricalDataForStockConsolidationCommand.
func NewAdjustHistoricalDataForStockConsolidationCommand(
	adjustHistoricalDataForStockConsolidationInteractor usecase.AdjustHistoricalDataForStockConsolidation,
) *AdjustHistoricalDataForStockConsolidationCommand {
	return &AdjustHistoricalDataForStockConsolidationCommand{
		adjustHistoricalDataForStockConsolidationInteractor: adjustHistoricalDataForStockConsolidationInteractor,
	}
}

// Command returns the command configuration.
func (c *AdjustHistoricalDataForStockConsolidationCommand) Command() *Command {
	return &Command{
		Name:   "adjust_historical_data_for_stock_consolidation",
		Usage:  "株式併合があった銘柄の過去の株価を調整する。",
		Action: c.Action,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "code",
				Usage:    "Code of the stock brand to adjust",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "consolidation-date",
				Usage:    "Date of the stock consolidation (format: YYYY-MM-DD)",
				Required: true,
			},
			&cli.Float64Flag{
				Name:     "consolidation-ratio",
				Usage:    "Consolidation ratio (旧株数/新株数。例: 5株を1株に併合する場合は 5.0)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Perform a dry run without saving changes",
			},
		},
	}
}

// Action is the action to be executed when the command is run.
func (c *AdjustHistoricalDataForStockConsolidationCommand) Action(ctx *cli.Context) error {
	code := ctx.String("code")
	consolidationDateStr := ctx.String("consolidation-date")
	consolidationRatioFloat := ctx.Float64("consolidation-ratio")
	dryRun := ctx.Bool("dry-run")

	consolidationDate, err := util.FormatStringToDate(consolidationDateStr)
	if err != nil {
		return errors.Wrap(err, "invalid consolidation-date format. use YYYY-MM-DD")
	}

	consolidationRatio := decimal.NewFromFloat(consolidationRatioFloat)
	if consolidationRatio.IsZero() || consolidationRatio.IsNegative() {
		return errors.New("consolidation-ratio must be positive")
	}

	err = c.adjustHistoricalDataForStockConsolidationInteractor.AdjustHistoricalDataForStockConsolidation(
		ctx.Context,
		code,
		consolidationDate,
		consolidationRatio,
		dryRun,
	)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

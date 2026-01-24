package commands

import (
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

// AdjustHistoricalDataForStockSplitCommand is a command to adjust historical data for stock splits.
type AdjustHistoricalDataForStockSplitCommand struct {
	adjustHistoricalDataForStockSplitInteractor usecase.AdjustHistoricalDataForStockSplit
}

// NewAdjustHistoricalDataForStockSplitCommand creates a new AdjustHistoricalDataForStockSplitCommand.
func NewAdjustHistoricalDataForStockSplitCommand(
	adjustHistoricalDataForStockSplitInteractor usecase.AdjustHistoricalDataForStockSplit,
) *AdjustHistoricalDataForStockSplitCommand {
	return &AdjustHistoricalDataForStockSplitCommand{
		adjustHistoricalDataForStockSplitInteractor: adjustHistoricalDataForStockSplitInteractor,
	}
}

// Command returns the command configuration.
func (c *AdjustHistoricalDataForStockSplitCommand) Command() *Command {
	return &Command{
		Name:   "adjust_historical_data_for_stock_split",
		Usage:  "株式分割があった銘柄の過去の株価を調整する。",
		Action: c.Action,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "code",
				Usage:    "Code of the stock brand to adjust",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "split-date",
				Usage:    "Date of the stock split (format: YYYY-MM-DD)",
				Required: true,
			},
			&cli.Float64Flag{
				Name:     "split-ratio",
				Usage:    "Split ratio (e.g., 2.0 for 1:2 split)",
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
func (c *AdjustHistoricalDataForStockSplitCommand) Action(ctx *cli.Context) error {
	code := ctx.String("code")
	splitDateStr := ctx.String("split-date")
	splitRatioFloat := ctx.Float64("split-ratio")
	dryRun := ctx.Bool("dry-run")

	splitDate, err := util.FormatStringToDate(splitDateStr)
	if err != nil {
		return errors.Wrap(err, "invalid split-date format. use YYYY-MM-DD")
	}

	splitRatio := decimal.NewFromFloat(splitRatioFloat)
	if splitRatio.IsZero() || splitRatio.IsNegative() {
		return errors.New("split-ratio must be positive")
	}

	err = c.adjustHistoricalDataForStockSplitInteractor.AdjustHistoricalDataForStockSplit(
		ctx.Context,
		code,
		splitDate,
		splitRatio,
		dryRun,
	)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

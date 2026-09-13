package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	c "github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/config"
	sContext "github.com/Code0716/stock-price-repository/context"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
	"github.com/Code0716/stock-price-repository/usecase"
)

type Runner struct {
	commands        []*commands.Command
	slackAPIClient  gateway.SlackAPIClient
	indexInteractor usecase.IndexInteractor
}

func NewRunner(
	healthCheckCommand *commands.HealthCheckCommand,
	updateStockBrandsInfoV1Command *commands.UpdateStockBrandsV1Command,
	createHistoricalDailyStockPricesV1Command *commands.CreateHistoricalDailyStockPricesV1Command,
	createDailyStockPriceV1Command *commands.CreateDailyStockPriceV1Command,
	rebuildAnalyzeDailyPricesV1Command *commands.RebuildAnalyzeDailyPricesV1Command,
	createNikkeiAndDjiHistoricalDataV1Command *commands.CreateNikkeiAndDjiHistoricalDataV1Command,
	adjustHistoricalDataForStockSplitCommand *commands.AdjustHistoricalDataForStockSplitCommand,
	adjustHistoricalDataForStockConsolidationCommand *commands.AdjustHistoricalDataForStockConsolidationCommand,
	exportYearlyDataCommand *commands.ExportYearlyDataCommand,
	exportMasterDataCommand *commands.ExportMasterDataCommand,
	syncFinAnnouncementsCommand *commands.SyncFinAnnouncementsCommand,
	syncFinStatementsCommand *commands.SyncFinStatementsCommand,
	backtestAllStocksCommand *commands.BacktestAllStocksCommand,
	syncFinStatementsAllStocksCommand *commands.SyncFinStatementsAllStocksCommand,
	gradeQuizAnswersV1Command *commands.GradeQuizAnswersV1Command,
	createQuizDailyUniverseV1Command *commands.CreateQuizDailyUniverseV1Command,
	evaluateDailyStockPicksV1Command *commands.EvaluateDailyStockPicksV1Command,
	createDailyStockPicksV1Command *commands.CreateDailyStockPicksV1Command,
	indexInteractor usecase.IndexInteractor,
	slackAPIClient gateway.SlackAPIClient,
) *Runner {
	r := &Runner{
		commands: []*commands.Command{
			healthCheckCommand.Command(),
			updateStockBrandsInfoV1Command.Command(),
			createHistoricalDailyStockPricesV1Command.Command(),
			createDailyStockPriceV1Command.Command(),
			rebuildAnalyzeDailyPricesV1Command.Command(),
			createNikkeiAndDjiHistoricalDataV1Command.Command(),
			adjustHistoricalDataForStockSplitCommand.Command(),
			adjustHistoricalDataForStockConsolidationCommand.Command(),
			exportYearlyDataCommand.Command(),
			exportMasterDataCommand.Command(),
			syncFinAnnouncementsCommand.Command(),
			syncFinStatementsCommand.Command(),
			backtestAllStocksCommand.Command(),
			syncFinStatementsAllStocksCommand.Command(),
			// grade_quiz_answers_v1 は create_daily_stock_price_v1 の後に実行すること（翌営業日終値の確定が前提）。
			gradeQuizAnswersV1Command.Command(),
			createQuizDailyUniverseV1Command.Command(),
			// evaluate_daily_stock_picks_v1 は create_daily_stock_price_v1 の後、
			// create_daily_stock_picks_v1 より先に実行すること（その日の答え合わせを早く確定させる）。
			evaluateDailyStockPicksV1Command.Command(),
			// create_daily_stock_picks_v1 も create_daily_stock_price_v1 の後に実行すること（当日引け値の確定が前提）。
			createDailyStockPicksV1Command.Command(),
		},
		indexInteractor: indexInteractor,
		slackAPIClient:  slackAPIClient,
	}
	return r
}

func (r *Runner) Run(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	commandName := args[1]

	ctx = sContext.SetTagName(ctx, commandName)

	app := c.NewApp()
	app.Commands = make([]*c.Command, 0, len(r.commands))
	for _, command := range r.commands {
		app.Commands = append(app.Commands, command.CliCommand())
	}

	start := time.Now()
	runErr := app.RunContext(ctx, args)
	elapsed := time.Since(start)

	params := resource.BatchNotificationParams{
		Source:      resource.BatchNotificationSourceSPR,
		Env:         config.GetApp().AppEnv,
		CommandName: commandName,
		Elapsed:     elapsed,
		FinishedAt:  time.Now(),
	}

	if runErr != nil {
		log.Printf("command: %s failed (elapsed=%v): %+v", commandName, elapsed, runErr)
		params.Err = runErr
		if slackErr := r.slackAPIClient.SendBlockMessage(
			ctx, gateway.SlackChannelNameDevNotification, resource.NewBatchFailureMessage(params),
		); slackErr != nil {
			// Block Kit 送信が落ちても失敗アラート自体は失わせず、プレーンテキストへ退避する。
			if fallbackErr := r.slackAPIClient.SendErrMessageNotification(
				ctx,
				errors.Wrap(runErr, fmt.Sprintf("Error command name: %s failed (block notify error: %v)", commandName, slackErr)),
			); fallbackErr != nil {
				return fallbackErr
			}
		}
		return runErr
	}

	if err := r.slackAPIClient.SendBlockMessage(
		ctx, gateway.SlackChannelNameDevNotification, resource.NewBatchSuccessMessage(params),
	); err != nil {
		if err := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(err, fmt.Sprintf("Error SendBlockMessage: %s failed.", commandName)),
		); err != nil {
			return err
		}
	}
	log.Printf("command: %s finished (elapsed=%v)", commandName, elapsed)
	return nil
}

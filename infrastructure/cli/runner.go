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
	createNikkeiAndDjiHistoricalDataV1Command *commands.CreateNikkeiAndDjiHistoricalDataV1Command,
	adjustHistoricalDataForStockSplitCommand *commands.AdjustHistoricalDataForStockSplitCommand,
	adjustHistoricalDataForStockConsolidationCommand *commands.AdjustHistoricalDataForStockConsolidationCommand,
	exportYearlyDataCommand *commands.ExportYearlyDataCommand,
	exportMasterDataCommand *commands.ExportMasterDataCommand,
	syncFinAnnouncementsCommand *commands.SyncFinAnnouncementsCommand,
	syncFinStatementsCommand *commands.SyncFinStatementsCommand,
	backtestAllStocksCommand *commands.BacktestAllStocksCommand,
	syncFinStatementsAllStocksCommand *commands.SyncFinStatementsAllStocksCommand,
	indexInteractor usecase.IndexInteractor,
	slackAPIClient gateway.SlackAPIClient,
) *Runner {
	r := &Runner{
		commands: []*commands.Command{
			healthCheckCommand.Command(),
			updateStockBrandsInfoV1Command.Command(),
			createHistoricalDailyStockPricesV1Command.Command(),
			createDailyStockPriceV1Command.Command(),
			createNikkeiAndDjiHistoricalDataV1Command.Command(),
			adjustHistoricalDataForStockSplitCommand.Command(),
			adjustHistoricalDataForStockConsolidationCommand.Command(),
			exportYearlyDataCommand.Command(),
			exportMasterDataCommand.Command(),
			syncFinAnnouncementsCommand.Command(),
			syncFinStatementsCommand.Command(),
			backtestAllStocksCommand.Command(),
			syncFinStatementsAllStocksCommand.Command(),
		},
		indexInteractor: indexInteractor,
		slackAPIClient:  slackAPIClient,
	}
	return r
}

// formatTimeTakenMessage は Slack 通知用の経過時間メッセージを生成する。
// 成功時・失敗時の両方で同じテンプレートを使い、運用ログでの統一感を保つ。
func formatTimeTakenMessage(commandName string, elapsed time.Duration) string {
	return fmt.Sprintf(
		"env: %s*\n*command name: %s*\n*time taken: %v",
		config.GetApp().AppEnv,
		commandName,
		elapsed,
	)
}

func (r *Runner) Run(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments")
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
	timeTakenMessage := formatTimeTakenMessage(commandName, elapsed)

	if runErr != nil {
		log.Printf("command: %s failed (elapsed=%v): %+v", commandName, elapsed, runErr)
		// エラー時にも経過時間を含めて Slack へ通知する。
		slackErr := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(runErr, fmt.Sprintf("Error command name: %s failed. %s", commandName, timeTakenMessage)),
		)
		if slackErr != nil {
			return slackErr
		}
		return runErr
	}

	if _, err := r.slackAPIClient.SendMessageByStrings(ctx, gateway.SlackChannelNameDevNotification, timeTakenMessage, nil, nil); err != nil {
		err := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(err, fmt.Sprintf("Error SendMessageByStrings: %s failed.", commandName)),
		)
		if err != nil {
			return err
		}
	}
	log.Printf("command: %s finished (elapsed=%v)", commandName, elapsed)
	return nil
}

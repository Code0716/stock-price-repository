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
	setJQuantsAPITokenToRedisV1Command *commands.SetJQuantsAPITokenToRedisV1Command,
	updateStockBrandsInfoV1Command *commands.UpdateStockBrandsV1Command,
	createHistoricalDailyStockPricesV1Command *commands.CreateHistoricalDailyStockPricesV1Command,
	createDailyStockPriceV1Command *commands.CreateDailyStockPriceV1Command,
	createNikkeiAndDjiHistoricalDataV1Command *commands.CreateNikkeiAndDjiHistoricalDataV1Command,
	exportStockBrandsAndDailyPriceToSQLV1Command *commands.ExportStockBrandsAndDailyPriceToSQLV1Command,
	indexInteractor usecase.IndexInteractor,
	slackAPIClient gateway.SlackAPIClient,
) *Runner {
	r := &Runner{
		commands: []*commands.Command{
			healthCheckCommand.Command(),
			setJQuantsAPITokenToRedisV1Command.Command(),
			updateStockBrandsInfoV1Command.Command(),
			createHistoricalDailyStockPricesV1Command.Command(),
			createDailyStockPriceV1Command.Command(),
			createNikkeiAndDjiHistoricalDataV1Command.Command(),
			exportStockBrandsAndDailyPriceToSQLV1Command.Command(),
		},
		indexInteractor: indexInteractor,
		slackAPIClient:  slackAPIClient,
	}
	return r
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
	msg := fmt.Sprintf("command: %s finished \n", commandName)
	if err := app.RunContext(ctx, args); err != nil {
		log.Print(msg, args, errors.Wrap(err, "command failed"))
		slackErr := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(err, fmt.Sprintf("Error command name: %s failed.", commandName)),
		)
		if slackErr != nil {
			return slackErr
		}
		return err
	}

	// FIXME: commandにかかった時間を通知する
	end := time.Now()
	timeTakenMessage := fmt.Sprintf(
		"env: %s*\n*command name: %s*\n*time taken: %v",
		config.App().AppEnv,
		commandName,
		end.Sub(start),
	)

	if _, err := r.slackAPIClient.SendMessageByStrings(ctx, gateway.SlackChannelNameDevNotification, timeTakenMessage, nil, nil); err != nil {
		err := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(err, fmt.Sprintf("Error SendMessageByStrings: %s failed.", commandName)),
		)
		if err != nil {
			return err
		}
	}
	log.Printf("%s command success", msg)
	return nil
}

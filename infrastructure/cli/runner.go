package cli

import (
	"context"
	"fmt"
	"log"
	"os"

	sContext "github.com/Code0716/stock-price-repository/context"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/pkg/errors"
	c "github.com/urfave/cli/v2"
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
	createNikkeiAndDjiHistoricalDataV1Command *commands.CreateNkkeiAndDjiHistoricalDataV1Command,
	exportStockBrandsAndDailyPriceToSQLV1Command *commands.ExportStockBrandsAndDailyPriceToSQLV1Command,
	indexInteractor usecase.IndexInteractor,
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
	}
	return r
}

func (r *Runner) Run() {
	args := os.Args

	ctx := context.Background()
	commandName := args[1]

	ctx = sContext.SetTagName(ctx, commandName)

	app := c.NewApp()
	app.Commands = make([]*c.Command, 0, len(r.commands))
	for _, command := range r.commands {
		app.Commands = append(app.Commands, command.CliCommand())
	}

	err := app.RunContext(ctx, args)
	msg := fmt.Sprintf("command: %s finished \n", commandName)
	if err != nil {
		log.Print(msg, args, errors.Wrap(err, "command failed"))
		err := r.slackAPIClient.SendErrMessageNotification(
			ctx,
			errors.Wrap(err, fmt.Sprintf("Error command name: %s failed.", commandName)),
		)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	log.Printf("%s command success", msg)
}

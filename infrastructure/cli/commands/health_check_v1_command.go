package commands

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
)

type HealthCheckCommand struct {
	slackAPIClient gateway.SlackAPIClient
}

func NewHealthCheckCommand(slackAPIClient gateway.SlackAPIClient) *HealthCheckCommand {
	return &HealthCheckCommand{slackAPIClient}
}

func (c *HealthCheckCommand) Command() *Command {
	return &Command{
		Name:   "health_check",
		Usage:  "動作確認用",
		Action: c.Action,
	}
}

func (c *HealthCheckCommand) Action(ctx *cli.Context) error {
	err := c.slackAPIClient.SendMessage(ctx.Context, gateway.SlackChannelNameDevNotification, resource.SlackMessageHealthCheck)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

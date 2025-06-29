package commands

import (
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/urfave/cli/v2"
)

type SetJQuantsAPITokenToRedisV1Command struct {
	stockAPIClient gateway.StockAPIClient
}

func NewSetJQuantsAPITokenToRedisV1Command(
	stockAPIClient gateway.StockAPIClient,
) *SetJQuantsAPITokenToRedisV1Command {
	return &SetJQuantsAPITokenToRedisV1Command{
		stockAPIClient,
	}
}

func (c *SetJQuantsAPITokenToRedisV1Command) Command() *Command {
	return &Command{
		Name:   "set_j_quants_api_token_to_redis_v1",
		Usage:  "j-Quantsのリフレッシュトークンを更新する",
		Action: c.Action,
	}
}

func (c *SetJQuantsAPITokenToRedisV1Command) Action(ctx *cli.Context) error {
	_, err := c.stockAPIClient.GetOrSetJQuantsAPIIDTokenToRedis(ctx.Context)
	return err
}

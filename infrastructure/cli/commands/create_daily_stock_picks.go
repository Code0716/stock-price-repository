package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// CreateDailyStockPicksV1Command create_daily_stock_picks_v1
// 最新営業日の引け値で翌営業日の買い候補をスクリーニングし、DB保存して Slack に通知する。
type CreateDailyStockPicksV1Command struct {
	interactor usecase.CreateDailyStockPicksInteractor
}

func NewCreateDailyStockPicksV1Command(interactor usecase.CreateDailyStockPicksInteractor) *CreateDailyStockPicksV1Command {
	return &CreateDailyStockPicksV1Command{interactor: interactor}
}

func (c *CreateDailyStockPicksV1Command) Command() *Command {
	return &Command{
		Name:  "create_daily_stock_picks_v1",
		Usage: "最新営業日の引け値で翌営業日の買い候補をスクリーニングし、DB保存して Slack に通知する。",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "top-n",
				Value: 25,
				Usage: "通知する銘柄数",
			},
			&cli.IntFlag{
				Name:  "max-per-sector",
				Value: 4,
				Usage: "同一33業種からの最大採用数（0で無制限）",
			},
			&cli.IntFlag{
				Name:  "concurrency",
				Value: 0,
				Usage: "ワーカー数（0 で CPU コア数）",
			},
			&cli.BoolFlag{
				Name:  "force",
				Value: false,
				Usage: "当日分が既にあっても作り直して再通知する",
			},
		},
		Action: c.Action,
	}
}

func (c *CreateDailyStockPicksV1Command) Action(ctx *cli.Context) error {
	err := c.interactor.CreateDailyStockPicks(
		ctx.Context,
		time.Now(),
		ctx.Int("top-n"),
		ctx.Int("max-per-sector"),
		ctx.Int("concurrency"),
		ctx.Bool("force"),
	)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

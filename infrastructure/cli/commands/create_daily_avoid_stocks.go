package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

// CreateDailyAvoidStocksV1Command create_daily_avoid_stocks_v1
// 流動性ユニバース内で直近12ヶ月の実現ボラティリティが上位20%に入る「避けるべき銘柄」を算出してDB保存する。
// 通知はしない（確認導線は front の /avoid-stocks のみ）。
type CreateDailyAvoidStocksV1Command struct {
	interactor usecase.CreateDailyAvoidStocksInteractor
}

func NewCreateDailyAvoidStocksV1Command(interactor usecase.CreateDailyAvoidStocksInteractor) *CreateDailyAvoidStocksV1Command {
	return &CreateDailyAvoidStocksV1Command{interactor: interactor}
}

func (c *CreateDailyAvoidStocksV1Command) Command() *Command {
	return &Command{
		Name:  "create_daily_avoid_stocks_v1",
		Usage: "流動性ユニバース内で直近12ヶ月ボラ上位20%の「避けるべき銘柄」を算出してDB保存する（通知はしない）。",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "date",
				Value: "",
				Usage: "判定基準日 YYYY-MM-DD（省略時は最新営業日。過去日バックフィル用）",
			},
			&cli.IntFlag{
				Name:  "concurrency",
				Value: 0,
				Usage: "ワーカー数（0 で CPU コア数）",
			},
			&cli.BoolFlag{
				Name:  "force",
				Value: false,
				Usage: "当日分が既にあっても作り直す",
			},
		},
		Action: c.Action,
	}
}

func (c *CreateDailyAvoidStocksV1Command) Action(ctx *cli.Context) error {
	var asOf *time.Time
	if d := ctx.String("date"); d != "" {
		parsed, err := util.FormatStringToDate(d)
		if err != nil {
			return errors.Wrap(err, "invalid --date")
		}
		asOf = &parsed
	}

	err := c.interactor.CreateDailyAvoidStocks(
		ctx.Context,
		time.Now(),
		asOf,
		ctx.Int("concurrency"),
		ctx.Bool("force"),
	)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

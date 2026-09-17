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
				Usage: "判定基準日 YYYY-MM-DD（省略時は最新営業日。--days とは併用不可）",
			},
			&cli.IntFlag{
				Name:  "days",
				Value: 0,
				Usage: "直近N営業日をまとめてバックフィルする（過去分の一括投入用。--date とは併用不可）",
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
	dateStr := ctx.String("date")
	days := ctx.Int("days")
	if dateStr != "" && days > 0 {
		return errors.New("--date と --days は併用できません")
	}

	if days > 0 {
		err := c.interactor.BackfillDailyAvoidStocks(
			ctx.Context,
			time.Now(),
			days,
			ctx.Int("concurrency"),
			ctx.Bool("force"),
		)
		if err != nil {
			return errors.Wrap(err, "Action error")
		}
		return nil
	}

	var asOf *time.Time
	if dateStr != "" {
		parsed, err := util.FormatStringToDate(dateStr)
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

package commands

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

type SyncFinStatementsAllStocksCommand struct {
	interactor usecase.StockBrandInteractor
}

func NewSyncFinStatementsAllStocksCommand(interactor usecase.StockBrandInteractor) *SyncFinStatementsAllStocksCommand {
	return &SyncFinStatementsAllStocksCommand{interactor: interactor}
}

func (c *SyncFinStatementsAllStocksCommand) Command() *Command {
	return &Command{
		Name:  "sync_fin_statements_all_stocks",
		Usage: "全主要市場銘柄の財務情報をj-Quantsから逐次取得してDBに保存する。週次バッチ用。",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "interval-ms",
				Value: 1100,
				Usage: "各リクエスト前のウェイトms。J-Quantsレート制限1req/sec対策",
			},
			&cli.IntFlag{
				Name:  "max",
				Value: 0,
				Usage: "処理する最大銘柄数（0=全件、テスト・検証用）",
			},
		},
		Action: c.Action,
	}
}

func (c *SyncFinStatementsAllStocksCommand) Action(ctx *cli.Context) error {
	if err := c.interactor.SyncFinStatementsAllStocks(ctx.Context, ctx.Int("interval-ms"), ctx.Int("max")); err != nil {
		return errors.Wrap(err, "SyncFinStatementsAllStocks error")
	}
	return nil
}

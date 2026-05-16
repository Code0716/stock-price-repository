package commands

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

type SyncFinStatementsCommand struct {
	stockBrandInteractor usecase.StockBrandInteractor
}

func NewSyncFinStatementsCommand(stockBrandInteractor usecase.StockBrandInteractor) *SyncFinStatementsCommand {
	return &SyncFinStatementsCommand{stockBrandInteractor}
}

func (c *SyncFinStatementsCommand) Command() *Command {
	return &Command{
		Name:  "sync_fin_statements",
		Usage: "j-Quantsから指定銘柄の財務情報を取得してDBに保存する。引数: --symbol=<証券コード>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "symbol",
				Usage:    "証券コード（例: 7203）",
				Required: true,
			},
		},
		Action: c.Action,
	}
}

func (c *SyncFinStatementsCommand) Action(ctx *cli.Context) error {
	symbol := ctx.String("symbol")
	if err := c.stockBrandInteractor.SyncFinStatements(ctx.Context, symbol); err != nil {
		return errors.Wrap(err, "SyncFinStatements error")
	}
	return nil
}

package commands

import (
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
)

// BacktestAllStocksCommand 全主要市場銘柄を全戦略でバックテストし、集計をRedisに保存するコマンド。
type BacktestAllStocksCommand struct {
	interactor usecase.StrategyRankingInteractor
}

func NewBacktestAllStocksCommand(interactor usecase.StrategyRankingInteractor) *BacktestAllStocksCommand {
	return &BacktestAllStocksCommand{interactor: interactor}
}

func (c *BacktestAllStocksCommand) Command() *Command {
	return &Command{
		Name:  "backtest_all_stocks_v1",
		Usage: "全主要市場銘柄を全戦略でバックテストし、戦略ランキングをRedisに保存する。",
		Flags: []cli.Flag{
			&cli.Float64Flag{
				Name:  "take-profit",
				Value: 0.10,
				Usage: "利確率（例: 0.10 = +10%）",
			},
			&cli.Float64Flag{
				Name:  "stop-loss",
				Value: 0.05,
				Usage: "損切り率（例: 0.05 = -5%）",
			},
			&cli.IntFlag{
				Name:  "max-hold-days",
				Value: 20,
				Usage: "最大保有営業日数",
			},
			&cli.IntFlag{
				Name:  "years",
				Value: 5,
				Usage: "バックテスト対象の直近N年",
			},
			&cli.IntFlag{
				Name:  "concurrency",
				Value: 0,
				Usage: "ワーカー数（0 で CPU コア数）",
			},
		},
		Action: c.Action,
	}
}

func (c *BacktestAllStocksCommand) Action(ctx *cli.Context) error {
	params := models.BacktestParams{
		TakeProfit:  decimal.NewFromFloat(ctx.Float64("take-profit")),
		StopLoss:    decimal.NewFromFloat(ctx.Float64("stop-loss")),
		MaxHoldDays: ctx.Int("max-hold-days"),
	}
	years := ctx.Int("years")
	concurrency := ctx.Int("concurrency")
	n, err := c.interactor.ComputeAndSaveStrategyRanking(ctx.Context, params, years, concurrency)
	if err != nil {
		return errors.Wrap(err, "ComputeAndSaveStrategyRanking error")
	}
	_ = n
	return nil
}

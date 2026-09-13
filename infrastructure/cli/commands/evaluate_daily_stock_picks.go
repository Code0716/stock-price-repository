package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// EvaluateDailyStockPicksV1Command evaluate_daily_stock_picks_v1
// 当日の日足取得後、推奨銘柄の1/3/5営業日後リターンと勝敗を確定させる。
type EvaluateDailyStockPicksV1Command struct {
	interactor usecase.EvaluateDailyStockPicksInteractor
}

func NewEvaluateDailyStockPicksV1Command(interactor usecase.EvaluateDailyStockPicksInteractor) *EvaluateDailyStockPicksV1Command {
	return &EvaluateDailyStockPicksV1Command{interactor: interactor}
}

func (c *EvaluateDailyStockPicksV1Command) Command() *Command {
	return &Command{
		Name:   "evaluate_daily_stock_picks_v1",
		Usage:  "当日の日足取得後、推奨銘柄の1/3/5営業日後リターンと勝敗を確定させる。",
		Action: c.Action,
	}
}

func (c *EvaluateDailyStockPicksV1Command) Action(ctx *cli.Context) error {
	err := c.interactor.EvaluateDailyStockPicks(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

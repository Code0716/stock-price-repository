package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// create_quiz_daily_universe_v1
type CreateQuizDailyUniverseV1Command struct {
	createQuizDailyUniverseInteractor usecase.CreateQuizDailyUniverseInteractor
}

func NewCreateQuizDailyUniverseV1Command(createQuizDailyUniverseInteractor usecase.CreateQuizDailyUniverseInteractor) *CreateQuizDailyUniverseV1Command {
	return &CreateQuizDailyUniverseV1Command{createQuizDailyUniverseInteractor}
}

func (c *CreateQuizDailyUniverseV1Command) Command() *Command {
	return &Command{
		Name:   "create_quiz_daily_universe_v1",
		Usage:  "当日の日足取得後、出来高がありよく動く300銘柄を選定しクイズの出題ユニバースを作成する。",
		Action: c.Action,
	}
}

func (c *CreateQuizDailyUniverseV1Command) Action(ctx *cli.Context) error {
	err := c.createQuizDailyUniverseInteractor.CreateQuizDailyUniverse(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}

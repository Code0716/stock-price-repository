package commands

import (
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

type CreateNikkeiAndDjiHistoricalDataV1Command struct {
	nikkeiInteractor usecase.IndexInteractor
}

func NewCreateNikkeiAndDjiHistoricalDataV1Command(
	nikkeiInteractor usecase.IndexInteractor,
) *CreateNikkeiAndDjiHistoricalDataV1Command {
	return &CreateNikkeiAndDjiHistoricalDataV1Command{nikkeiInteractor}
}

func (c *CreateNikkeiAndDjiHistoricalDataV1Command) Command() *Command {
	return &Command{
		Name:   "create_nikkei_and_dji_historical_data_v1",
		Usage:  "日経平均、NYダウの日足を最大限保存する",
		Action: c.Action,
	}
}

func (c *CreateNikkeiAndDjiHistoricalDataV1Command) Action(ctx *cli.Context) error {
	err := c.nikkeiInteractor.CreateNikkeiAndDjiHistoricalData(ctx.Context, time.Now())
	if err != nil {
		return errors.Wrap(err, "nikkeiInteractor.CreateNikkeiAndDjiHistoricalData error")
	}

	return nil
}

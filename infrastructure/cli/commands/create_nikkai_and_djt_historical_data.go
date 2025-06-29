package commands

import (
	"time"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

type CreateNkkeiAndDjiHistoricalDataV1Command struct {
	nikkeiInteractor usecase.IndexInteractor
}

func NewCreateNkkeiAndDjiHistoricalDataV1Command(
	nikkeiInteractor usecase.IndexInteractor,
) *CreateNkkeiAndDjiHistoricalDataV1Command {
	return &CreateNkkeiAndDjiHistoricalDataV1Command{nikkeiInteractor}
}

func (c *CreateNkkeiAndDjiHistoricalDataV1Command) Command() *Command {
	return &Command{
		Name:   "create_nikkei_and_dji_historical_data_v1",
		Usage:  "日経平均、NYダウの日足を最大限保存する",
		Action: c.Action,
	}
}

func (c *CreateNkkeiAndDjiHistoricalDataV1Command) Action(ctx *cli.Context) error {
	now := time.Now()
	err := c.nikkeiInteractor.CreateNikkeiAndDjiHistoricalData(ctx.Context, now)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

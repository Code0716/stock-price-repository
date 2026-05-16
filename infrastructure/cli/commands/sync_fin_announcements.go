package commands

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

type SyncFinAnnouncementsCommand struct {
	stockBrandInteractor usecase.StockBrandInteractor
}

func NewSyncFinAnnouncementsCommand(stockBrandInteractor usecase.StockBrandInteractor) *SyncFinAnnouncementsCommand {
	return &SyncFinAnnouncementsCommand{stockBrandInteractor}
}

func (c *SyncFinAnnouncementsCommand) Command() *Command {
	return &Command{
		Name:  "sync_fin_announcements",
		Usage: "j-Quantsから決算発表予定を取得してDBに保存する。",
		Action: c.Action,
	}
}

func (c *SyncFinAnnouncementsCommand) Action(ctx *cli.Context) error {
	if err := c.stockBrandInteractor.SyncFinAnnouncements(ctx.Context); err != nil {
		return errors.Wrap(err, "SyncFinAnnouncements error")
	}
	return nil
}

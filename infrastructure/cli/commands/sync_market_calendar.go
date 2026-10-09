package commands

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/usecase"
)

type SyncMarketCalendarCommand struct {
	syncMarketCalendarInteractor usecase.SyncMarketCalendarInteractor
}

func NewSyncMarketCalendarCommand(syncMarketCalendarInteractor usecase.SyncMarketCalendarInteractor) *SyncMarketCalendarCommand {
	return &SyncMarketCalendarCommand{syncMarketCalendarInteractor}
}

func (c *SyncMarketCalendarCommand) Command() *Command {
	return &Command{
		Name:  "sync_market_calendar",
		Usage: "取引カレンダー(j-Quants)と市場イベント(SQ・日銀・FOMC・米CPI・米雇用統計)を取得してDBに保存する。",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "manual",
				Usage: "手入力イベントのYAMLパス。省略時は MARKET_EVENTS_MANUAL_PATH（既定 resources/market_events_manual.yaml）。ファイルが無ければ無視する。",
			},
		},
		Action: c.Action,
	}
}

func (c *SyncMarketCalendarCommand) Action(ctx *cli.Context) error {
	manualYAML, err := readManualEvents(manualEventsPath(ctx.String("manual")))
	if err != nil {
		return errors.Wrap(err, "readManualEvents error")
	}

	if err := c.syncMarketCalendarInteractor.SyncMarketCalendar(ctx.Context, time.Now(), manualYAML); err != nil {
		return errors.Wrap(err, "SyncMarketCalendar error")
	}
	return nil
}

func manualEventsPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return config.GetEconomicCalendar().ManualEventsPath
}

// readManualEvents 手入力YAMLを読む。ファイルが無い場合は nil（手入力なし）を返す。
func readManualEvents(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "os.ReadFile error: "+path)
	}
	return b, nil
}

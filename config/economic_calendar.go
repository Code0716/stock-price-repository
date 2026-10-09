package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// EconomicCalendar 市場イベント（日銀・FOMC・米CPI/雇用統計）の取得元URLと手入力ファイルの設定。
type EconomicCalendar struct {
	BOJScheduleURL  string `envconfig:"boj_schedule_url" default:"https://www.boj.or.jp/mopo/mpmsche_minu/index.htm"`
	FOMCCalendarURL string `envconfig:"fomc_calendar_url" default:"https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm"`
	BLSICSURL       string `envconfig:"bls_ics_url" default:"https://www.bls.gov/schedule/news_release/bls.ics"`
	// HTTPUserAgent BLS は User-Agent 無しのリクエストを拒否するため必須。
	HTTPUserAgent string `envconfig:"economic_calendar_user_agent" default:"stock-price-repository/1.0 (market calendar sync)"`
	// ManualEventsPath 手入力で上書きするイベントの YAML。ファイルが無ければ無視する。
	ManualEventsPath string `envconfig:"market_events_manual_path" default:"resources/market_events_manual.yaml"`
}

var economicCalendar EconomicCalendar

func LoadConfigEconomicCalendar() {
	prefix := ""
	err := envconfig.Process(prefix, &economicCalendar)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetEconomicCalendar() *EconomicCalendar {
	return &economicCalendar
}

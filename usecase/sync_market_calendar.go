//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/usecase/marketcal"
)

// marketCalendarFetchTimeout 外部サイト1リクエストあたりのタイムアウト（共有 HTTP クライアントはタイムアウトを持たないため）。
const marketCalendarFetchTimeout = 30 * time.Second

type SyncMarketCalendarInteractor interface {
	// SyncMarketCalendar 取引カレンダーと市場イベント（SQ・日銀・FOMC・米CPI・米雇用統計）を同期する。
	//   - 取引カレンダー: J-Quants から「now の1年前〜翌年末」を取得して upsert する。
	//   - 市場イベント: now の年と翌年を対象に、(種別, 年) 単位で洗い替える。
	//   - manualYAML: 手入力 YAML。ある (種別, 年) は取得結果を捨てて手入力で置き換える。nil なら使わない。
	// 失敗したソースの (種別, 年) は更新せず既存データを残し、他のソースは続行して、最後にまとめて error を返す。
	SyncMarketCalendar(ctx context.Context, now time.Time, manualYAML []byte) error
}

type syncMarketCalendarInteractorImpl struct {
	tx                       repositories.Transaction
	stockAPIClient           gateway.StockAPIClient
	economicCalendarClient   gateway.EconomicCalendarClient
	marketCalendarRepository repositories.MarketCalendarRepository
}

func NewSyncMarketCalendarInteractor(
	tx repositories.Transaction,
	stockAPIClient gateway.StockAPIClient,
	economicCalendarClient gateway.EconomicCalendarClient,
	marketCalendarRepository repositories.MarketCalendarRepository,
) SyncMarketCalendarInteractor {
	return &syncMarketCalendarInteractorImpl{
		tx:                       tx,
		stockAPIClient:           stockAPIClient,
		economicCalendarClient:   economicCalendarClient,
		marketCalendarRepository: marketCalendarRepository,
	}
}

func (si *syncMarketCalendarInteractorImpl) SyncMarketCalendar(ctx context.Context, now time.Time, manualYAML []byte) error {
	var failures []string
	fail := func(source string, err error) {
		failures = append(failures, source+": "+err.Error())
	}

	fromYear, toYear := now.Year(), now.Year()+1
	eventsFrom := time.Date(fromYear, time.January, 1, 0, 0, 0, 0, time.Local)
	eventsTo := time.Date(toYear, time.December, 31, 0, 0, 0, 0, time.Local)

	var auto []*models.MarketEvent

	// 取引カレンダー → SQ
	days, err := si.syncTradingCalendar(ctx, now.AddDate(-1, 0, 0), eventsTo)
	if err != nil {
		fail("取引カレンダー(j-quants)", err)
	} else {
		auto = append(auto, marketcal.CalcSQ(days, eventsFrom, eventsTo)...)
	}

	// 日銀
	if events, err := si.fetchMeetings(ctx, si.economicCalendarClient.FetchBOJSchedulePage, marketcal.ParseBOJ, fromYear, toYear); err != nil {
		fail("日銀会合(boj_html)", err)
	} else {
		auto = append(auto, events...)
	}

	// FOMC
	if events, err := si.fetchMeetings(ctx, si.economicCalendarClient.FetchFOMCPage, marketcal.ParseFOMC, fromYear, toYear); err != nil {
		fail("FOMC(fed_html)", err)
	} else {
		auto = append(auto, events...)
	}

	// 米CPI・米雇用統計
	if events, err := si.fetchBLS(ctx, fromYear, toYear); err != nil {
		fail("米CPI/雇用統計(bls_ics)", err)
	} else {
		auto = append(auto, events...)
	}

	// 手入力（取得に失敗したソースの復旧用）
	var manual []*models.MarketEvent
	if len(manualYAML) > 0 {
		manual, err = marketcal.ParseManualEvents(manualYAML)
		if err != nil {
			fail("手入力YAML(manual)", err)
			manual = nil
		}
	}

	grouped := marketcal.ApplyManualOverride(auto, manual)
	if err := si.replaceEvents(ctx, grouped); err != nil {
		fail("市場イベント保存", err)
	}

	if len(failures) > 0 {
		return errors.Errorf("SyncMarketCalendar: %d件のソースで失敗しました（失敗したソースの既存データは維持）: %s", len(failures), strings.Join(failures, " / "))
	}
	return nil
}

// syncTradingCalendar J-Quants から取引カレンダーを取得して upsert し、取得した日を返す。
func (si *syncMarketCalendarInteractorImpl) syncTradingCalendar(ctx context.Context, from, to time.Time) ([]*models.MarketCalendarDay, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, marketCalendarFetchTimeout)
	defer cancel()

	res, err := si.stockAPIClient.GetTradingCalendar(fetchCtx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "GetTradingCalendar error")
	}
	if len(res) == 0 {
		return nil, errors.New("取引カレンダーが0件です")
	}

	days := make([]*models.MarketCalendarDay, 0, len(res))
	for _, r := range res {
		days = append(days, &models.MarketCalendarDay{Date: r.Date, HolDiv: r.HolDiv})
	}
	if err := si.marketCalendarRepository.UpsertDays(ctx, days); err != nil {
		return nil, errors.Wrap(err, "UpsertDays error")
	}
	return days, nil
}

// fetchMeetings 日銀・FOMC のページを取得・パースし、年8回であることを確かめた対象年のイベントを返す。
func (si *syncMarketCalendarInteractorImpl) fetchMeetings(
	ctx context.Context,
	fetch func(context.Context) ([]byte, error),
	parse func([]byte) ([]*models.MarketEvent, error),
	fromYear, toYear int,
) ([]*models.MarketEvent, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, marketCalendarFetchTimeout)
	defer cancel()

	body, err := fetch(fetchCtx)
	if err != nil {
		return nil, errors.Wrap(err, "fetch error")
	}
	events, err := parse(body)
	if err != nil {
		return nil, errors.Wrap(err, "parse error")
	}
	return marketcal.ValidateAnnualMeetings(events, fromYear, toYear)
}

func (si *syncMarketCalendarInteractorImpl) fetchBLS(ctx context.Context, fromYear, toYear int) ([]*models.MarketEvent, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, marketCalendarFetchTimeout)
	defer cancel()

	body, err := si.economicCalendarClient.FetchBLSICS(fetchCtx)
	if err != nil {
		return nil, errors.Wrap(err, "fetch error")
	}
	events, err := marketcal.ParseBLSICS(body)
	if err != nil {
		return nil, errors.Wrap(err, "parse error")
	}
	events = marketcal.FilterYears(events, fromYear, toYear)
	if len(events) == 0 {
		return nil, errors.Errorf("%d〜%d年のイベントがありません", fromYear, toYear)
	}
	return events, nil
}

// replaceEvents (種別, 年) 単位で、その年の同種別イベントを洗い替える。全体を1トランザクションにする。
func (si *syncMarketCalendarInteractorImpl) replaceEvents(ctx context.Context, grouped map[marketcal.KindYear][]*models.MarketEvent) error {
	if len(grouped) == 0 {
		return nil
	}
	err := si.tx.DoInTx(ctx, func(ctx context.Context) error {
		for _, k := range marketcal.SortedKeys(grouped) {
			from := time.Date(k.Year, time.January, 1, 0, 0, 0, 0, time.Local)
			to := time.Date(k.Year, time.December, 31, 0, 0, 0, 0, time.Local)
			if err := si.marketCalendarRepository.ReplaceEvents(ctx, k.Kind, from, to, grouped[k]); err != nil {
				return errors.Wrapf(err, "ReplaceEvents error kind=%s year=%d", k.Kind, k.Year)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}
	return nil
}

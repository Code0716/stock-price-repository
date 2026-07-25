//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

// dailyStockPickDatesDefaultLimit 日付一覧の既定件数。
const dailyStockPickDatesDefaultLimit = 90

// DailyStockPickInteractor 買い候補の閲覧用（読み取り専用）ユースケース。
// 書き込みバッチ（CreateDailyStockPicksInteractor / EvaluateDailyStockPicksInteractor）とは
// 別インターフェースにする。バッチ側は Slack 依存を持つため API から使い回さない。
type DailyStockPickInteractor interface {
	// GetDay 指定 pick_date の推奨一覧とサマリを返す。date が nil なら最新 pick_date を使う。
	// 該当日が無い場合は PickDate=nil / Items=[] を返す（エラーにしない）。
	GetDay(ctx context.Context, date *time.Time) (*models.DailyStockPickDay, error)
	// GetPickDates pick_date を新しい順に最大 limit 件返す。limit<=0 なら既定値を使う。
	GetPickDates(ctx context.Context, limit int) (*models.DailyStockPickDates, error)
	// GetStats 期間とスコアバージョンで絞った累計成績を返す。scoreVersion が空なら現行バージョンを使う。
	GetStats(ctx context.Context, from, to *time.Time, scoreVersion string) (*models.DailyStockPickStats, error)
}

type dailyStockPickInteractorImpl struct {
	dailyStockPickRepository repositories.DailyStockPickRepository
	stockBrandRepository     repositories.StockBrandRepository
}

func NewDailyStockPickInteractor(
	dailyStockPickRepository repositories.DailyStockPickRepository,
	stockBrandRepository repositories.StockBrandRepository,
) DailyStockPickInteractor {
	return &dailyStockPickInteractorImpl{
		dailyStockPickRepository: dailyStockPickRepository,
		stockBrandRepository:     stockBrandRepository,
	}
}

func (di *dailyStockPickInteractorImpl) GetDay(ctx context.Context, date *time.Time) (*models.DailyStockPickDay, error) {
	pickDate, err := di.resolvePickDate(ctx, date)
	if err != nil {
		return nil, errors.Wrap(err, "resolvePickDate error")
	}
	empty := &models.DailyStockPickDay{
		ScoreVersion: domain_service.DailyPickScoreVersion,
		Items:        []*models.DailyStockPickItem{},
	}
	if pickDate == nil {
		// 1件も推奨が無い（バッチ未実行）。エラーにせず空で返す。
		return empty, nil
	}

	picks, err := di.dailyStockPickRepository.ListByPickDate(ctx, *pickDate)
	if err != nil {
		return nil, errors.Wrap(err, "ListByPickDate error")
	}
	if len(picks) == 0 {
		// 休場日など、指定日にデータが無いケース。
		return empty, nil
	}

	names, err := di.resolveBrandNames(ctx, picks)
	if err != nil {
		return nil, err
	}

	items := make([]*models.DailyStockPickItem, 0, len(picks))
	for _, p := range picks {
		items = append(items, toDailyStockPickItem(p, names[p.StockBrandID]))
	}

	dateStr := picks[0].PickDate.Format(util.DateLayout)
	day := &models.DailyStockPickDay{
		PickDate:     &dateStr,
		ScoreVersion: picks[0].ScoreVersion,
		Summary:      domain_service.SummarizeDailyPicksForView(picks),
		Items:        items,
	}
	day.Evaluated = day.Summary.PendingCount == 0
	if picks[0].NotifiedAt != nil {
		notifiedAt := picks[0].NotifiedAt.Format(time.RFC3339)
		day.NotifiedAt = &notifiedAt
	}
	return day, nil
}

func (di *dailyStockPickInteractorImpl) GetPickDates(ctx context.Context, limit int) (*models.DailyStockPickDates, error) {
	if limit <= 0 {
		limit = dailyStockPickDatesDefaultLimit
	}

	dates, err := di.dailyStockPickRepository.ListPickDates(ctx, limit)
	if err != nil {
		return nil, errors.Wrap(err, "ListPickDates error")
	}

	out := make([]string, 0, len(dates))
	for _, d := range dates {
		out = append(out, d.Format(util.DateLayout))
	}
	return &models.DailyStockPickDates{Dates: out}, nil
}

func (di *dailyStockPickInteractorImpl) GetStats(ctx context.Context, from, to *time.Time, scoreVersion string) (*models.DailyStockPickStats, error) {
	if scoreVersion == "" {
		scoreVersion = domain_service.DailyPickScoreVersion
	}

	picks, err := di.dailyStockPickRepository.ListByDateRange(ctx, from, to, scoreVersion)
	if err != nil {
		return nil, errors.Wrap(err, "ListByDateRange error")
	}

	stats := &models.DailyStockPickStats{
		ScoreVersion: scoreVersion,
		Totals:       domain_service.SummarizeDailyPicksForView(picks),
		Daily:        domain_service.AggregateDailyPicksByDate(picks),
		ScoreBands:   domain_service.AggregateDailyPicksByScoreBand(picks),
	}
	// from/to は実データの範囲を返す（クエリ未指定でも軸が分かるようにする）。
	if len(stats.Daily) > 0 {
		first := stats.Daily[0].PickDate
		last := stats.Daily[len(stats.Daily)-1].PickDate
		stats.From = &first
		stats.To = &last
	}
	return stats, nil
}

// resolvePickDate date が nil なら最新 pick_date を引く。1件も無ければ nil を返す。
func (di *dailyStockPickInteractorImpl) resolvePickDate(ctx context.Context, date *time.Time) (*time.Time, error) {
	if date != nil {
		return date, nil
	}
	return di.dailyStockPickRepository.FindLatestPickDate(ctx)
}

// resolveBrandNames 推奨銘柄の名称を stock_brand から解決する（Name は daily_stock_pick に保存していないため）。
func (di *dailyStockPickInteractorImpl) resolveBrandNames(ctx context.Context, picks []*models.DailyStockPick) (map[string]string, error) {
	ids := make([]string, 0, len(picks))
	seen := make(map[string]struct{}, len(picks))
	for _, p := range picks {
		if _, ok := seen[p.StockBrandID]; ok {
			continue
		}
		seen[p.StockBrandID] = struct{}{}
		ids = append(ids, p.StockBrandID)
	}

	brands, err := di.stockBrandRepository.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.Wrap(err, "FindByIDs error")
	}

	names := make(map[string]string, len(brands))
	for _, b := range brands {
		names[b.ID] = b.Name
	}
	return names, nil
}

// toDailyStockPickItem ドメインモデルを API レスポンス用に詰め替える。
func toDailyStockPickItem(p *models.DailyStockPick, name string) *models.DailyStockPickItem {
	item := &models.DailyStockPickItem{
		PickRank:         p.PickRank,
		StockBrandID:     p.StockBrandID,
		TickerSymbol:     p.TickerSymbol,
		Name:             name,
		Score:            p.Score,
		ScoreVersion:     p.ScoreVersion,
		SignalCount:      p.SignalCount,
		Strategies:       toDailyStockPickStrategies(p.Strategies),
		Sector33CodeName: p.Sector33CodeName,
		BaseClosePrice:   p.BaseClosePrice,
		AvgTradingValue:  p.AvgTradingValue,
		VolumeRatio:      p.VolumeRatio,
		ADX:              p.ADX,
		PlusDI:           p.PlusDI,
		MinusDI:          p.MinusDI,
		ATRRatio:         p.ATRRatio,
		RSI:              p.RSI,
		Return1D:         p.Return1D,
		Return3D:         p.Return3D,
		Return5D:         p.Return5D,
		Outcome:          p.Outcome,
	}
	if p.EvaluatedAt != nil {
		evaluatedAt := p.EvaluatedAt.Format(time.RFC3339)
		item.EvaluatedAt = &evaluatedAt
	}
	return item
}

// toDailyStockPickStrategies 戦略キーに日本語表示名を添える（フロントに日本語辞書を重複させないため）。
// 未知のキーはラベルにキーをそのまま使う。
func toDailyStockPickStrategies(keys []string) []*models.DailyStockPickStrategy {
	out := make([]*models.DailyStockPickStrategy, 0, len(keys))
	for _, k := range keys {
		label, ok := domain_service.StrategyLabels[k]
		if !ok {
			label = k
		}
		out = append(out, &models.DailyStockPickStrategy{Key: k, Label: label})
	}
	return out
}

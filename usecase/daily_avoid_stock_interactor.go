//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

// dailyAvoidStockDatesDefaultLimit 日付一覧の既定件数。
const dailyAvoidStockDatesDefaultLimit = 90

// DailyAvoidStockInteractor 避けるべき銘柄の閲覧用（読み取り専用）ユースケース。
// 書き込みバッチ（CreateDailyAvoidStocksInteractor）とは別インターフェースにする。
type DailyAvoidStockInteractor interface {
	// GetDay 指定 as_of_date の避けるべき銘柄一覧とサマリを返す。date が nil なら最新 as_of_date を使う。
	// 該当日が無い場合は AsOfDate=nil / Items=[] を返す（エラーにしない）。
	GetDay(ctx context.Context, date *time.Time) (*models.DailyAvoidStockDay, error)
	// GetAsOfDates as_of_date を新しい順に最大 limit 件返す。limit<=0 なら既定値を使う。
	GetAsOfDates(ctx context.Context, limit int) (*models.DailyAvoidStockDates, error)
}

type dailyAvoidStockInteractorImpl struct {
	dailyAvoidStockRepository repositories.DailyAvoidStockRepository
	stockBrandRepository      repositories.StockBrandRepository
}

func NewDailyAvoidStockInteractor(
	dailyAvoidStockRepository repositories.DailyAvoidStockRepository,
	stockBrandRepository repositories.StockBrandRepository,
) DailyAvoidStockInteractor {
	return &dailyAvoidStockInteractorImpl{
		dailyAvoidStockRepository: dailyAvoidStockRepository,
		stockBrandRepository:      stockBrandRepository,
	}
}

func (di *dailyAvoidStockInteractorImpl) GetDay(ctx context.Context, date *time.Time) (*models.DailyAvoidStockDay, error) {
	asOfDate, err := di.resolveAsOfDate(ctx, date)
	if err != nil {
		return nil, errors.Wrap(err, "resolveAsOfDate error")
	}
	empty := &models.DailyAvoidStockDay{
		RuleVersion:         domain_service.DailyAvoidRuleVersion,
		ThresholdVolatility: decimal.Zero,
		Items:               []*models.DailyAvoidStockItem{},
	}
	if asOfDate == nil {
		// 1件も無い（バッチ未実行）。エラーにせず空で返す。
		return empty, nil
	}

	rows, err := di.dailyAvoidStockRepository.ListByAsOfDate(ctx, *asOfDate)
	if err != nil {
		return nil, errors.Wrap(err, "ListByAsOfDate error")
	}
	if len(rows) == 0 {
		// 休場日など、指定日にデータが無いケース。
		return empty, nil
	}

	names, err := di.resolveBrandNames(ctx, rows)
	if err != nil {
		return nil, err
	}

	items := make([]*models.DailyAvoidStockItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toDailyAvoidStockItem(r, names[r.StockBrandID]))
	}

	dateStr := rows[0].AsOfDate.Format(util.DateLayout)
	return &models.DailyAvoidStockDay{
		AsOfDate:            &dateStr,
		RuleVersion:         rows[0].RuleVersion,
		UniverseSize:        rows[0].UniverseSize,
		ThresholdVolatility: rows[0].ThresholdVolatility,
		Summary:             domain_service.SummarizeDailyAvoidStocks(rows),
		Items:               items,
	}, nil
}

func (di *dailyAvoidStockInteractorImpl) GetAsOfDates(ctx context.Context, limit int) (*models.DailyAvoidStockDates, error) {
	if limit <= 0 {
		limit = dailyAvoidStockDatesDefaultLimit
	}

	dates, err := di.dailyAvoidStockRepository.ListAsOfDates(ctx, limit)
	if err != nil {
		return nil, errors.Wrap(err, "ListAsOfDates error")
	}

	out := make([]string, 0, len(dates))
	for _, d := range dates {
		out = append(out, d.Format(util.DateLayout))
	}
	return &models.DailyAvoidStockDates{Dates: out}, nil
}

// resolveAsOfDate date が nil なら最新 as_of_date を引く。1件も無ければ nil を返す。
func (di *dailyAvoidStockInteractorImpl) resolveAsOfDate(ctx context.Context, date *time.Time) (*time.Time, error) {
	if date != nil {
		return date, nil
	}
	return di.dailyAvoidStockRepository.FindLatestAsOfDate(ctx)
}

// resolveBrandNames 銘柄名を stock_brand から解決する（Name は daily_avoid_stock に保存していないため）。
func (di *dailyAvoidStockInteractorImpl) resolveBrandNames(ctx context.Context, rows []*models.DailyAvoidStock) (map[string]string, error) {
	ids := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.StockBrandID]; ok {
			continue
		}
		seen[r.StockBrandID] = struct{}{}
		ids = append(ids, r.StockBrandID)
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

// toDailyAvoidStockItem ドメインモデルを API レスポンス用に詰め替える。
func toDailyAvoidStockItem(r *models.DailyAvoidStock, name string) *models.DailyAvoidStockItem {
	return &models.DailyAvoidStockItem{
		AvoidRank:            r.AvoidRank,
		StockBrandID:         r.StockBrandID,
		TickerSymbol:         r.TickerSymbol,
		Name:                 name,
		Severity:             r.Severity,
		Reason:               r.Reason,
		Volatility12M:        r.Volatility12M,
		VolatilityPercentile: r.VolatilityPercentile,
		AvgTradingValue:      r.AvgTradingValue,
		BaseClosePrice:       r.BaseClosePrice,
		Sector33CodeName:     r.Sector33CodeName,
	}
}

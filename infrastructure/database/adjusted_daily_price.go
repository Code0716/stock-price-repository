//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// AdjustedDailyPriceRepositoryImpl は stock_brands_daily_price_for_analyze を読む
// AdjustedDailyPriceRepository の実装。クエリの構成は
// StockBrandsDailyPriceRepositoryImpl（stock_brands_daily_price 向け）の写しで、
// テーブルを for_analyze に差し替えたもの。
type AdjustedDailyPriceRepositoryImpl struct {
	query *genQuery.Query
}

func NewAdjustedDailyPriceRepositoryImpl(db *gorm.DB) repositories.AdjustedDailyPriceRepository {
	return &AdjustedDailyPriceRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (si *AdjustedDailyPriceRepositoryImpl) GetLatestPriceBySymbol(ctx context.Context, symbol string) (*models.StockBrandDailyPrice, error) {
	tx := TxOrDefault(ctx, si.query)
	price, err := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).
		Where(tx.StockBrandsDailyPriceForAnalyze.TickerSymbol.Eq(symbol)).
		Order(tx.StockBrandsDailyPriceForAnalyze.Date.Desc()).
		First()
	if err != nil {
		return nil, errors.Wrap(err, "AdjustedDailyPriceRepository.GetLatestPriceBySymbol error")
	}

	return si.convertToDomainModel(price), nil
}

func (si *AdjustedDailyPriceRepositoryImpl) ListDailyPricesBySymbol(ctx context.Context, filter models.ListDailyPricesBySymbolFilter) ([]*models.StockBrandDailyPrice, error) {
	tx := TxOrDefault(ctx, si.query)

	if filter.TickerSymbol == "" {
		return nil, errors.New("TickerSymbol is required")
	}
	q := tx.WithContext(ctx).
		StockBrandsDailyPriceForAnalyze.
		Where(
			tx.StockBrandsDailyPriceForAnalyze.
				TickerSymbol.
				Eq(filter.TickerSymbol),
		)

	if filter.DateFrom != nil {
		dateOnlyFrom := time.Date(
			filter.DateFrom.Year(),
			filter.DateFrom.Month(),
			filter.DateFrom.Day(),
			0, 0, 0, 0,
			filter.DateFrom.Location(),
		)
		q = q.Where(tx.StockBrandsDailyPriceForAnalyze.Date.Gte(dateOnlyFrom))
	}

	if filter.DateTo != nil {
		dateOnlyTo := time.Date(
			filter.DateTo.Year(),
			filter.DateTo.Month(),
			filter.DateTo.Day(),
			0, 0, 0, 0,
			filter.DateTo.Location(),
		)
		q = q.Where(tx.StockBrandsDailyPriceForAnalyze.Date.Lte(dateOnlyTo))
	}

	// ソート順の適用（デフォルトは昇順）
	if filter.DateOrder != nil && *filter.DateOrder == models.SortOrderDesc {
		q = q.Order(tx.StockBrandsDailyPriceForAnalyze.Date.Desc())
	} else {
		q = q.Order(tx.StockBrandsDailyPriceForAnalyze.Date)
	}

	rdbDailyPrices, err := q.Find()
	if err != nil {
		return nil, errors.Wrap(err, "AdjustedDailyPriceRepository.ListDailyPricesBySymbol error")
	}

	domainDailyPrices := make([]*models.StockBrandDailyPrice, 0, len(rdbDailyPrices))
	for _, rdbDailyPrice := range rdbDailyPrices {
		domainDailyPrices = append(domainDailyPrices, si.convertToDomainModel(rdbDailyPrice))
	}

	return domainDailyPrices, nil
}

// ListRangePricesBySymbols 複数銘柄の期間中日足を一括取得する（シグナル精度評価用）
func (si *AdjustedDailyPriceRepositoryImpl) ListRangePricesBySymbols(ctx context.Context, filter models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error) {
	tx := TxOrDefault(ctx, si.query)

	if len(filter.Symbols) == 0 {
		return []*models.StockBrandDailyPrice{}, nil
	}

	q := tx.WithContext(ctx).
		StockBrandsDailyPriceForAnalyze.
		Where(tx.StockBrandsDailyPriceForAnalyze.TickerSymbol.In(filter.Symbols...))

	if filter.DateFrom != nil {
		dateOnly := time.Date(filter.DateFrom.Year(), filter.DateFrom.Month(), filter.DateFrom.Day(), 0, 0, 0, 0, filter.DateFrom.Location())
		q = q.Where(tx.StockBrandsDailyPriceForAnalyze.Date.Gte(dateOnly))
	}
	if filter.DateTo != nil {
		dateOnly := time.Date(filter.DateTo.Year(), filter.DateTo.Month(), filter.DateTo.Day(), 0, 0, 0, 0, filter.DateTo.Location())
		q = q.Where(tx.StockBrandsDailyPriceForAnalyze.Date.Lte(dateOnly))
	}

	q = q.Order(tx.StockBrandsDailyPriceForAnalyze.TickerSymbol).Order(tx.StockBrandsDailyPriceForAnalyze.Date)

	rows, err := q.Find()
	if err != nil {
		return nil, errors.Wrap(err, "AdjustedDailyPriceRepository.ListRangePricesBySymbols error")
	}

	prices := make([]*models.StockBrandDailyPrice, 0, len(rows))
	for _, r := range rows {
		prices = append(prices, si.convertToDomainModel(r))
	}
	return prices, nil
}

// ListRecentTradingDates onOrBefore以前の直近の営業日（データが存在する日）を新しい順にlimit件取得する（クイズのユニバース選定用）。
func (si *AdjustedDailyPriceRepositoryImpl) ListRecentTradingDates(ctx context.Context, onOrBefore time.Time, limit int) ([]time.Time, error) {
	tx := TxOrDefault(ctx, si.query)

	dateOnly := time.Date(onOrBefore.Year(), onOrBefore.Month(), onOrBefore.Day(), 0, 0, 0, 0, onOrBefore.Location())

	var dates []time.Time
	if err := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).
		Where(tx.StockBrandsDailyPriceForAnalyze.Date.Lte(dateOnly)).
		Distinct(tx.StockBrandsDailyPriceForAnalyze.Date).
		Order(tx.StockBrandsDailyPriceForAnalyze.Date.Desc()).
		Limit(limit).
		Pluck(tx.StockBrandsDailyPriceForAnalyze.Date, &dates); err != nil {
		return nil, errors.Wrap(err, "AdjustedDailyPriceRepository.ListRecentTradingDates error")
	}

	return dates, nil
}

// analyzeDailyPriceWithBrandIDRow - for_analyzeとstock_brandをJOINした行。
// stock_brands_daily_price_for_analyzeの列（GORMタグで束縛）に加え、
// stock_brand.id をエイリアス stock_brand_id として受け取る。
type analyzeDailyPriceWithBrandIDRow struct {
	ID            string    `gorm:"column:id"`
	TickerSymbol  string    `gorm:"column:ticker_symbol"`
	Date          time.Time `gorm:"column:date"`
	OpenPrice     float64   `gorm:"column:open_price"`
	ClosePrice    float64   `gorm:"column:close_price"`
	HighPrice     float64   `gorm:"column:high_price"`
	LowPrice      float64   `gorm:"column:low_price"`
	AdjClosePrice float64   `gorm:"column:adj_close_price"`
	Volume        uint64    `gorm:"column:volume"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
	StockBrandID  string    `gorm:"column:stock_brand_id"`
}

// ListPricesByDateRange 期間中の全銘柄の日足を取得する（クイズのユニバース選定用）。
// for_analyzeにはstock_brand_idが無いため、stock_brandをticker_symbolでJOINして埋める
// （クイズの主要市場フィルタがStockBrandIDで判定するため必須）。
func (si *AdjustedDailyPriceRepositoryImpl) ListPricesByDateRange(ctx context.Context, from, to time.Time) ([]*models.StockBrandDailyPrice, error) {
	tx := TxOrDefault(ctx, si.query)

	dateFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	dateTo := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())

	var rows []*analyzeDailyPriceWithBrandIDRow
	result := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).UnderlyingDB().Raw(`
		SELECT a.*, b.id AS stock_brand_id
		FROM stock_brands_daily_price_for_analyze a
		INNER JOIN stock_brand b ON b.ticker_symbol = a.ticker_symbol AND b.deleted_at IS NULL
		WHERE a.date >= ? AND a.date <= ?
		ORDER BY a.ticker_symbol, a.date
	`, dateFrom, dateTo).Scan(&rows)
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "AdjustedDailyPriceRepository.ListPricesByDateRange error")
	}

	prices := make([]*models.StockBrandDailyPrice, 0, len(rows))
	for _, r := range rows {
		prices = append(prices, &models.StockBrandDailyPrice{
			ID:           r.ID,
			StockBrandID: r.StockBrandID,
			TickerSymbol: r.TickerSymbol,
			Date:         r.Date,
			Open:         decimal.NewFromFloat(r.OpenPrice),
			Close:        decimal.NewFromFloat(r.ClosePrice),
			High:         decimal.NewFromFloat(r.HighPrice),
			Low:          decimal.NewFromFloat(r.LowPrice),
			Adjclose:     decimal.NewFromFloat(r.AdjClosePrice),
			Volume:       int64(r.Volume),
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}
	return prices, nil
}

// FindNextTradingDate afterより後の直近の営業日を1件取得する（存在しなければnil。クイズ採点用）。
func (si *AdjustedDailyPriceRepositoryImpl) FindNextTradingDate(ctx context.Context, after time.Time) (*time.Time, error) {
	tx := TxOrDefault(ctx, si.query)

	dateOnly := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location())

	price, err := tx.StockBrandsDailyPriceForAnalyze.WithContext(ctx).
		Where(tx.StockBrandsDailyPriceForAnalyze.Date.Gt(dateOnly)).
		Order(tx.StockBrandsDailyPriceForAnalyze.Date).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "AdjustedDailyPriceRepository.FindNextTradingDate error")
	}

	return &price.Date, nil
}

func (si *AdjustedDailyPriceRepositoryImpl) convertToDomainModel(dailyPriceDB *genModel.StockBrandsDailyPriceForAnalyze) *models.StockBrandDailyPrice {
	if dailyPriceDB == nil {
		return nil
	}

	return &models.StockBrandDailyPrice{
		ID:           dailyPriceDB.ID,
		TickerSymbol: dailyPriceDB.TickerSymbol,
		Date:         dailyPriceDB.Date,
		Open:         decimal.NewFromFloat(dailyPriceDB.OpenPrice),
		Close:        decimal.NewFromFloat(dailyPriceDB.ClosePrice),
		High:         decimal.NewFromFloat(dailyPriceDB.HighPrice),
		Low:          decimal.NewFromFloat(dailyPriceDB.LowPrice),
		Adjclose:     decimal.NewFromFloat(dailyPriceDB.AdjClosePrice),
		Volume:       int64(dailyPriceDB.Volume),
		CreatedAt:    dailyPriceDB.CreatedAt,
		UpdatedAt:    dailyPriceDB.UpdatedAt,
		// StockBrandID は for_analyze に列が無いため空文字のまま。
		// クイズの ListPricesByDateRange のみ別途 stock_brand を JOIN して埋めている。
	}
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type AnalyzeStockBrandPriceHistoryRepositoryImpl struct {
	query *genQuery.Query
	db    *gorm.DB
}

func NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db *gorm.DB) repositories.AnalyzeStockBrandPriceHistoryRepository {
	return &AnalyzeStockBrandPriceHistoryRepositoryImpl{
		query: genQuery.Use(db),
		db:    db,
	}
}

type analyzeStockBrandPriceHistoryRow struct {
	ID           string
	StockBrandID string
	Name         string
	TickerSymbol string
	TradePrice   float64
	CurrentPrice float64
	Action       string
	Method       string
	Memo         *string
	CreatedAt    time.Time
}

func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) buildWhereQuery(db *gorm.DB, filter *models.AnalyzeStockBrandPriceHistoryFilter) *gorm.DB {
	if filter.TickerSymbol != "" {
		db = db.Where("h.ticker_symbol = ?", filter.TickerSymbol)
	}
	if filter.Action != "" {
		db = db.Where("h.action = ?", filter.Action)
	}
	if filter.Method != "" {
		db = db.Where("h.method = ?", filter.Method)
	}
	return db
}

func orderClause(sortBy, order string) string {
	dir := "DESC"
	if order == models.AnalyzeStockBrandPriceHistoryOrderAsc {
		dir = "ASC"
	}

	switch sortBy {
	case models.AnalyzeStockBrandPriceHistorySortByProfit:
		return fmt.Sprintf("(COALESCE(d.close_price, h.trade_price) - h.trade_price) %s, h.id %s", dir, dir)
	case models.AnalyzeStockBrandPriceHistorySortByProfitRate:
		return fmt.Sprintf("((COALESCE(d.close_price, h.trade_price) - h.trade_price) / h.trade_price) %s, h.id %s", dir, dir)
	default:
		return fmt.Sprintf("h.created_at %s, h.id %s", dir, dir)
	}
}

// FindWithFilter 条件に一致する分析履歴を取得する
// current_price は stock_brands_daily_price の最新 close_price を JOIN して都度算出する
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) FindWithFilter(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) ([]*models.AnalyzeStockBrandPriceHistory, error) {
	db := ai.db.WithContext(ctx).
		Table("analyze_stock_brand_price_history AS h").
		Select(`
			h.id,
			h.stock_brand_id,
			COALESCE(s.name, '') AS name,
			h.ticker_symbol,
			h.trade_price,
			COALESCE(d.close_price, h.trade_price) AS current_price,
			h.action,
			h.method,
			h.memo,
			h.created_at
		`).
		Joins("LEFT JOIN stock_brand AS s ON s.id = h.stock_brand_id AND s.deleted_at IS NULL").
		Joins(`LEFT JOIN stock_brands_daily_price AS d ON d.id = (
			SELECT id FROM stock_brands_daily_price
			WHERE ticker_symbol = h.ticker_symbol AND deleted_at IS NULL
			ORDER BY date DESC LIMIT 1
		)`)

	db = ai.buildWhereQuery(db, filter)
	db = db.Order(orderClause(filter.SortBy, filter.Order))

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	db = db.Limit(limit).Offset(offset)

	var rows []*analyzeStockBrandPriceHistoryRow
	if err := db.Find(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.FindWithFilter error")
	}

	histories := make([]*models.AnalyzeStockBrandPriceHistory, 0, len(rows))
	for _, row := range rows {
		histories = append(histories, models.NewAnalyzeStockBrandPriceHistory(
			row.ID,
			row.StockBrandID,
			row.Name,
			row.TickerSymbol,
			decimal.NewFromFloat(row.TradePrice),
			decimal.NewFromFloat(row.CurrentPrice),
			row.Action,
			row.Method,
			row.Memo,
			row.CreatedAt,
		))
	}

	return histories, nil
}

// CountWithFilter 条件に一致する分析履歴の総件数を返す
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) CountWithFilter(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (int64, error) {
	db := ai.db.WithContext(ctx).
		Table("analyze_stock_brand_price_history AS h")

	db = ai.buildWhereQuery(db, filter)

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.CountWithFilter error")
	}

	return count, nil
}

// FindMultipleSignals 同一日に2つ以上のシグナルが出た銘柄を集計して返す
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) FindMultipleSignals(ctx context.Context, filter *models.MultipleSignalStockFilter) ([]*models.MultipleSignalStock, error) {
	type multipleSignalRow struct {
		StockBrandID string
		Name         string
		TickerSymbol string
		Date         time.Time
		Methods      string
		SignalCount  int
		CurrentPrice float64
	}

	var dateCondition string
	args := make([]interface{}, 0, 3)

	if filter.Date != nil {
		dateCondition = "DATE(h.created_at) = DATE(?)"
		args = append(args, *filter.Date)
	} else {
		dateCondition = "DATE(h.created_at) = (SELECT DATE(MAX(created_at)) FROM analyze_stock_brand_price_history)"
	}

	whereClause := dateCondition
	if filter.Cursor != "" {
		whereClause += " AND h.ticker_symbol > ?"
		args = append(args, filter.Cursor)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT
			h.stock_brand_id,
			ANY_VALUE(COALESCE(s.name, '')) AS name,
			h.ticker_symbol,
			DATE(h.created_at) AS date,
			GROUP_CONCAT(DISTINCT h.method ORDER BY h.method ASC SEPARATOR ',') AS methods,
			COUNT(DISTINCT h.method) AS signal_count,
			ANY_VALUE(COALESCE(d.close_price, 0)) AS current_price
		FROM analyze_stock_brand_price_history h
		LEFT JOIN stock_brand AS s ON s.id = h.stock_brand_id AND s.deleted_at IS NULL
		LEFT JOIN stock_brands_daily_price AS d ON d.id = (
			SELECT id FROM stock_brands_daily_price
			WHERE ticker_symbol = h.ticker_symbol AND deleted_at IS NULL
			ORDER BY date DESC LIMIT 1
		)
		WHERE %s
		GROUP BY h.stock_brand_id, h.ticker_symbol, DATE(h.created_at)
		HAVING COUNT(DISTINCT h.method) >= 2
		ORDER BY h.ticker_symbol ASC
		LIMIT ?
	`, whereClause)

	var rows []*multipleSignalRow
	if err := ai.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.FindMultipleSignals error")
	}

	stocks := make([]*models.MultipleSignalStock, 0, len(rows))
	for _, r := range rows {
		var methods []string
		if r.Methods != "" {
			methods = strings.Split(r.Methods, ",")
		}
		stocks = append(stocks, &models.MultipleSignalStock{
			StockBrandID: r.StockBrandID,
			Name:         r.Name,
			TickerSymbol: r.TickerSymbol,
			Date:         r.Date,
			Methods:      methods,
			SignalCount:  r.SignalCount,
			CurrentPrice: decimal.NewFromFloat(r.CurrentPrice),
		})
	}

	return stocks, nil
}

// DeleteByStockBrandIDs 銘柄IDと一致したものを削除する
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) DeleteByStockBrandIDs(ctx context.Context, ids []string) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = ai.query
	}

	if _, err := tx.AnalyzeStockBrandPriceHistory.WithContext(ctx).
		Where(tx.AnalyzeStockBrandPriceHistory.StockBrandID.In(ids...)).
		Delete(); err != nil {
		return errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.DeleteByStockBrandIDs error")
	}

	return nil
}

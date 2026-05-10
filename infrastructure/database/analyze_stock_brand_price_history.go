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

	if filter.TickerSymbol != "" {
		db = db.Where("h.ticker_symbol = ?", filter.TickerSymbol)
	}
	if filter.Action != "" {
		db = db.Where("h.action = ?", filter.Action)
	}
	if filter.Method != "" {
		db = db.Where("h.method = ?", filter.Method)
	}
	if filter.Cursor != "" {
		var cursorRow struct {
			CreatedAt time.Time
			ID        string
		}
		if err := ai.db.WithContext(ctx).
			Table("analyze_stock_brand_price_history").
			Select("created_at, id").
			Where("id = ?", filter.Cursor).
			Take(&cursorRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.FindWithFilter cursor error")
		}
		db = db.Where("(h.created_at < ? OR (h.created_at = ? AND h.id < ?))", cursorRow.CreatedAt, cursorRow.CreatedAt, cursorRow.ID)
	}

	db = db.Order("h.created_at DESC").Order("h.id DESC")
	if filter.Limit > 0 {
		db = db.Limit(filter.Limit)
	}

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

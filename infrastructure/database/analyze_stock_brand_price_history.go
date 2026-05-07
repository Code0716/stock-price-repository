//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
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

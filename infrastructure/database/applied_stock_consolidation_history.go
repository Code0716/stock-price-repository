package database

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	"github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type AppliedStockConsolidationsHistoryRepositoryImpl struct {
	query *gen_query.Query
}

func NewAppliedStockConsolidationsHistoryRepositoryImpl(db *gorm.DB) repositories.AppliedStockConsolidationsHistoryRepository {
	return &AppliedStockConsolidationsHistoryRepositoryImpl{
		query: gen_query.Use(db),
	}
}

func (r *AppliedStockConsolidationsHistoryRepositoryImpl) Exists(ctx context.Context, symbol string, consolidationDate time.Time) (bool, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	count, err := tx.AppliedStockConsolidationsHistory.WithContext(ctx).
		Where(tx.AppliedStockConsolidationsHistory.Symbol.Eq(symbol)).
		Where(tx.AppliedStockConsolidationsHistory.ConsolidationDate.Eq(consolidationDate)).
		Count()
	return count > 0, err
}

func (r *AppliedStockConsolidationsHistoryRepositoryImpl) Create(ctx context.Context, history *models.AppliedStockConsolidationHistory) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	ratioVal, _ := history.Ratio.Float64()

	dbModel := &gen_model.AppliedStockConsolidationsHistory{
		Symbol:            history.Symbol,
		ConsolidationDate: history.ConsolidationDate,
		Ratio:             ratioVal,
	}
	return tx.AppliedStockConsolidationsHistory.WithContext(ctx).Create(dbModel)
}

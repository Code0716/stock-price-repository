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

type AppliedStockSplitsHistoryRepositoryImpl struct {
	query *gen_query.Query
}

func NewAppliedStockSplitsHistoryRepositoryImpl(db *gorm.DB) repositories.AppliedStockSplitsHistoryRepository {
	return &AppliedStockSplitsHistoryRepositoryImpl{
		query: gen_query.Use(db),
	}
}

func (r *AppliedStockSplitsHistoryRepositoryImpl) Exists(ctx context.Context, symbol string, splitDate time.Time) (bool, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	count, err := tx.AppliedStockSplitsHistory.WithContext(ctx).
		Where(tx.AppliedStockSplitsHistory.Symbol.Eq(symbol)).
		Where(tx.AppliedStockSplitsHistory.SplitDate.Eq(splitDate)).
		Count()
	return count > 0, err
}

func (r *AppliedStockSplitsHistoryRepositoryImpl) Create(ctx context.Context, history *models.AppliedStockSplitHistory) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	ratioVal, _ := history.Ratio.Float64()

	dbModel := &gen_model.AppliedStockSplitsHistory{
		Symbol:    history.Symbol,
		SplitDate: history.SplitDate,
		Ratio:     ratioVal,
	}
	return tx.AppliedStockSplitsHistory.WithContext(ctx).Create(dbModel)
}

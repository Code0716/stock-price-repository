package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
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
	tx := TxOrDefault(ctx, r.query)

	count, err := tx.AppliedStockSplitsHistory.WithContext(ctx).
		Where(tx.AppliedStockSplitsHistory.Symbol.Eq(symbol)).
		Where(tx.AppliedStockSplitsHistory.SplitDate.Eq(splitDate)).
		Count()
	return count > 0, err
}

// TruncateAll 全件を物理削除する。5年再構築バッチ専用の破壊的操作。
func (r *AppliedStockSplitsHistoryRepositoryImpl) TruncateAll(ctx context.Context) error {
	tx := TxOrDefault(ctx, r.query)

	if err := tx.AppliedStockSplitsHistory.WithContext(ctx).UnderlyingDB().
		Exec("TRUNCATE TABLE applied_stock_splits_history").Error; err != nil {
		return errors.Wrap(err, "AppliedStockSplitsHistory.TruncateAll error")
	}
	return nil
}

func (r *AppliedStockSplitsHistoryRepositoryImpl) Create(ctx context.Context, history *models.AppliedStockSplitHistory) error {
	tx := TxOrDefault(ctx, r.query)

	ratioVal, _ := history.Ratio.Float64()

	dbModel := &gen_model.AppliedStockSplitsHistory{
		Symbol:    history.Symbol,
		SplitDate: history.SplitDate,
		Ratio:     ratioVal,
	}
	return tx.AppliedStockSplitsHistory.WithContext(ctx).Create(dbModel)
}

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

type AppliedStockConsolidationsHistoryRepositoryImpl struct {
	query *gen_query.Query
}

func NewAppliedStockConsolidationsHistoryRepositoryImpl(db *gorm.DB) repositories.AppliedStockConsolidationsHistoryRepository {
	return &AppliedStockConsolidationsHistoryRepositoryImpl{
		query: gen_query.Use(db),
	}
}

func (r *AppliedStockConsolidationsHistoryRepositoryImpl) Exists(ctx context.Context, symbol string, consolidationDate time.Time) (bool, error) {
	tx := TxOrDefault(ctx, r.query)

	count, err := tx.AppliedStockConsolidationsHistory.WithContext(ctx).
		Where(tx.AppliedStockConsolidationsHistory.Symbol.Eq(symbol)).
		Where(tx.AppliedStockConsolidationsHistory.ConsolidationDate.Eq(consolidationDate)).
		Count()
	return count > 0, err
}

// TruncateAll 全件を物理削除する。5年再構築バッチ専用の破壊的操作。
func (r *AppliedStockConsolidationsHistoryRepositoryImpl) TruncateAll(ctx context.Context) error {
	tx := TxOrDefault(ctx, r.query)

	if err := tx.AppliedStockConsolidationsHistory.WithContext(ctx).UnderlyingDB().
		Exec("TRUNCATE TABLE applied_stock_consolidations_history").Error; err != nil {
		return errors.Wrap(err, "AppliedStockConsolidationsHistory.TruncateAll error")
	}
	return nil
}

func (r *AppliedStockConsolidationsHistoryRepositoryImpl) Create(ctx context.Context, history *models.AppliedStockConsolidationHistory) error {
	tx := TxOrDefault(ctx, r.query)

	ratioVal, _ := history.Ratio.Float64()

	dbModel := &gen_model.AppliedStockConsolidationsHistory{
		Symbol:            history.Symbol,
		ConsolidationDate: history.ConsolidationDate,
		Ratio:             ratioVal,
	}
	return tx.AppliedStockConsolidationsHistory.WithContext(ctx).Create(dbModel)
}

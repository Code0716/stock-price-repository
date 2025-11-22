//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type AnalyzeStockBrandPriceHistoryRepositoryImpl struct {
	query *genQuery.Query
}

func NewAnalyzeStockBrandPriceHistoryRepositoryImpl(db *gorm.DB) repositories.AnalyzeStockBrandPriceHistoryRepository {
	return &AnalyzeStockBrandPriceHistoryRepositoryImpl{
		query: genQuery.Use(db),
	}
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

// CreateOrUpdate 銘柄の価格をupdateする
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) CreateOrUpdate(ctx context.Context, histories []*models.AnalyzeStockBrandPriceHistory) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = ai.query
	}
	err := tx.AnalyzeStockBrandPriceHistory.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stock_brand_id"}},
			DoUpdates: clause.AssignmentColumns(
				[]string{
					"current_price",
				}),
		}).Create(ai.convertToDBModels(histories)...)
	if err != nil {
		return errors.Wrap(err, "AnalyzeStockBrandPriceHistoryRepositoryImpl.CreateOrUpdate error")
	}

	return nil
}

// convertToDBModels converts a slice of models to a slice of database models.
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) convertToDBModels(histories []*models.AnalyzeStockBrandPriceHistory) []*genModel.AnalyzeStockBrandPriceHistory {
	var analyzeStockBrandPriceHistoryDB []*genModel.AnalyzeStockBrandPriceHistory
	for _, v := range histories {
		analyzeStockBrandPriceHistoryDB = append(analyzeStockBrandPriceHistoryDB, ai.convertToDBModel(v))
	}
	return analyzeStockBrandPriceHistoryDB
}

// convertToDBModel converts a model to a database model.
func (ai *AnalyzeStockBrandPriceHistoryRepositoryImpl) convertToDBModel(histories *models.AnalyzeStockBrandPriceHistory) *genModel.AnalyzeStockBrandPriceHistory {
	tradePrice, _ := histories.TradePrice.Round(4).Float64()
	currentPrice, _ := histories.CurrentPrice.Round(4).Float64()

	return &genModel.AnalyzeStockBrandPriceHistory{
		ID:           histories.ID,
		StockBrandID: histories.StockBrandID,
		TickerSymbol: histories.TickerSymbol,
		TradePrice:   tradePrice,
		CurrentPrice: currentPrice,
		Action:       histories.Action,
		Method:       histories.Method,
		Memo:         histories.Memo,
		CreatedAt:    &histories.CreatedAt,
	}
}

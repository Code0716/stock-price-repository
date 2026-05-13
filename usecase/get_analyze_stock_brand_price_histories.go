package usecase

import (
	"context"
	"math"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

// GetAnalyzeStockBrandPriceHistories 分析履歴一覧を取得する
func (si *stockBrandInteractorImpl) GetAnalyzeStockBrandPriceHistories(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (*models.PaginatedAnalyzeStockBrandPriceHistories, error) {
	if filter == nil {
		filter = &models.AnalyzeStockBrandPriceHistoryFilter{}
	}

	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	total, err := si.analyzeStockBrandPriceHistoryRepository.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "分析履歴件数の取得に失敗しました")
	}

	histories, err := si.analyzeStockBrandPriceHistoryRepository.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "分析履歴一覧の取得に失敗しました")
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &models.PaginatedAnalyzeStockBrandPriceHistories{
		Histories:  histories,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

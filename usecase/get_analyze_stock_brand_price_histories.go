package usecase

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

// GetAnalyzeStockBrandPriceHistories 分析履歴一覧を取得する
func (si *stockBrandInteractorImpl) GetAnalyzeStockBrandPriceHistories(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (*models.PaginatedAnalyzeStockBrandPriceHistories, error) {
	if filter == nil {
		filter = &models.AnalyzeStockBrandPriceHistoryFilter{}
	}

	limit := filter.Limit
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	repoFilter := *filter
	repoFilter.Limit = fetchLimit

	histories, err := si.analyzeStockBrandPriceHistoryRepository.FindWithFilter(ctx, &repoFilter)
	if err != nil {
		return nil, errors.Wrap(err, "分析履歴一覧の取得に失敗しました")
	}

	result := &models.PaginatedAnalyzeStockBrandPriceHistories{
		Histories: histories,
		Limit:     limit,
	}

	if limit > 0 && len(histories) > limit {
		nextHistory := histories[limit]
		result.NextCursor = &nextHistory.ID
		result.Histories = histories[:limit]
	}

	return result, nil
}

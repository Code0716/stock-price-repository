package usecase

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

// GetMultipleSignalStocks 同一日に2つ以上のシグナルが出た銘柄一覧を取得する
func (si *stockBrandInteractorImpl) GetMultipleSignalStocks(ctx context.Context, filter *models.MultipleSignalStockFilter) (*models.PaginatedMultipleSignalStocks, error) {
	if filter == nil {
		filter = &models.MultipleSignalStockFilter{}
	}

	limit := filter.Limit
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	repoFilter := *filter
	repoFilter.Limit = fetchLimit

	stocks, err := si.analyzeStockBrandPriceHistoryRepository.FindMultipleSignals(ctx, &repoFilter)
	if err != nil {
		return nil, errors.Wrap(err, "複数シグナル銘柄一覧の取得に失敗しました")
	}

	result := &models.PaginatedMultipleSignalStocks{
		Stocks: stocks,
		Limit:  limit,
	}

	if limit > 0 && len(stocks) > limit {
		nextStock := stocks[limit]
		result.NextCursor = &nextStock.TickerSymbol
		result.Stocks = stocks[:limit]
	}

	return result, nil
}

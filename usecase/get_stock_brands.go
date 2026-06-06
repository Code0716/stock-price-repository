//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

// GetStockBrands 銘柄一覧を取得する
// keyword: 銘柄コード前方一致 OR 銘柄名部分一致フィルタ（空文字列の場合はフィルタなし）
// symbolFrom: ページネーション用の開始シンボル（inclusive、空文字列の場合は最初から）
// limit: 取得件数上限
// onlyMainMarkets: true の場合、マーケットコード 111, 112, 113 のみを取得
func (si *stockBrandInteractorImpl) GetStockBrands(ctx context.Context, keyword string, symbolFrom string, limit int, onlyMainMarkets bool) (*models.PaginatedStockBrands, error) {
	// limitが指定されている場合、次ページの有無を判定するため+1件取得
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	filter := models.NewStockBrandFilter().WithPagination(symbolFrom, fetchLimit)
	if keyword != "" {
		filter = filter.WithKeyword(keyword)
	}
	if onlyMainMarkets {
		filter = filter.WithOnlyMainMarkets()
	}

	brands, err := si.stockBrandRepository.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "銘柄一覧の取得に失敗しました")
	}

	// ページネーション情報の構築
	result := &models.PaginatedStockBrands{
		Brands: brands,
		Limit:  limit,
	}

	// limitが指定されている場合、次ページの有無を判定
	if limit > 0 && len(brands) > limit {
		// limit+1件取得しているので、limit件を超えていれば次ページあり
		// 次のページの最初の銘柄（limit番目）のシンボルをカーソルに設定
		nextBrand := brands[limit]
		result.NextCursor = &nextBrand.TickerSymbol
		result.Brands = brands[:limit]
	}

	return result, nil
}

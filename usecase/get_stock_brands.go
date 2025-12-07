//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

// GetStockBrands 銘柄一覧を取得する
// symbolFrom: 取得開始シンボル（ページネーション用）
// limit: 取得件数上限
// onlyMainMarkets: true の場合、マーケットコード 111, 112, 113 のみを取得
func (si *stockBrandInteractorImpl) GetStockBrands(ctx context.Context, symbolFrom string, limit int, onlyMainMarkets bool) (*models.PaginatedStockBrands, error) {
	var brands []*models.StockBrand
	var err error

	// limitが指定されている場合、次ページの有無を判定するため+1件取得
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	if onlyMainMarkets {
		// 主要市場のみ取得（DB層でフィルタリング）
		if symbolFrom != "" && fetchLimit > 0 {
			brands, err = si.stockBrandRepository.FindFromSymbolMainMarkets(ctx, symbolFrom, fetchLimit)
		} else {
			brands, err = si.stockBrandRepository.FindAllMainMarkets(ctx)
		}
	} else {
		// 全市場取得
		if symbolFrom != "" && fetchLimit > 0 {
			brands, err = si.stockBrandRepository.FindFromSymbol(ctx, symbolFrom, fetchLimit)
		} else {
			brands, err = si.stockBrandRepository.FindAll(ctx)
		}
	}

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
		lastBrand := brands[limit-1]
		result.NextCursor = &lastBrand.TickerSymbol
		result.Brands = brands[:limit]
	}

	return result, nil
}

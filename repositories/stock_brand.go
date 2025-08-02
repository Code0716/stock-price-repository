//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type StockBrandRepository interface {
	// 銘柄をupsertする。
	UpsertStockBrands(ctx context.Context, stockBrands []*models.StockBrand) error
	// 銘柄を全件取得する。
	FindAll(ctx context.Context) ([]*models.StockBrand, error)
	// シンボルから昇順に上場銘柄を取得する。
	FindFromSymbol(ctx context.Context, symbolFrom string, limit int) ([]*models.StockBrand, error)
	// 上場廃止銘柄の取得
	// upsertされたタイミングで利用。upsertされてなかったら上場廃止と判断する
	FindDelistingStockBrandsFromUpdateTime(ctx context.Context, now time.Time) ([]string, error)
	// 上場廃止銘柄を削除する。
	DeleteDelistingStockBrands(ctx context.Context, ids []string) ([]*models.StockBrand, error)
}

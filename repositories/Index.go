//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type NikkeiRepository interface {
	CreateNikkeiStockAverageDailyPrices(ctx context.Context, averageDailyPrices models.IndexStockAverageDailyPrices) error
	// ListNikkeiStockAverageDailyPrices 日付範囲（両端 nil 可）で日経平均日足を昇順取得する。
	ListNikkeiStockAverageDailyPrices(ctx context.Context, from, to *time.Time) (models.IndexStockAverageDailyPrices, error)
}

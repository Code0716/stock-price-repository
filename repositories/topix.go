//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type TopixRepository interface {
	CreateTopixDailyPrices(ctx context.Context, dailyPrices models.IndexStockAverageDailyPrices) error
	// ListTopixDailyPrices 日付範囲（両端 nil 可）で TOPIX 日足を昇順取得する。
	ListTopixDailyPrices(ctx context.Context, from, to *time.Time) (models.IndexStockAverageDailyPrices, error)
}

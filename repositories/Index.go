//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type NikkeiRepository interface {
	CreateNikkeiStockAverageDailyPrices(ctx context.Context, averageDailyPrices models.IndexStockAverageDailyPrices) error
}

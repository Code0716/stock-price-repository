//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type AppliedStockSplitsHistoryRepository interface {
	Exists(ctx context.Context, symbol string, splitDate time.Time) (bool, error)
	Create(ctx context.Context, history *models.AppliedStockSplitHistory) error
}

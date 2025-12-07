//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

// HighVolumeStockBrandRepository provides access to high volume stock brand data
type HighVolumeStockBrandRepository interface {
	// FindAll retrieves all high volume stock brands
	FindAll(ctx context.Context) ([]*models.HighVolumeStockBrand, error)

	// FindWithPagination retrieves high volume stock brands with cursor-based pagination
	FindWithPagination(ctx context.Context, symbolFrom string, limit int) ([]*models.HighVolumeStockBrand, error)
}

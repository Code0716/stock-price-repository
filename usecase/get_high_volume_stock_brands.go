//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// GetHighVolumeStockBrandsUseCase handles business logic for retrieving high volume stock brands
type GetHighVolumeStockBrandsUseCase interface {
	Execute(ctx context.Context) ([]*models.HighVolumeStockBrand, error)
	ExecuteWithPagination(ctx context.Context, symbolFrom string, limit int) (*models.PaginatedHighVolumeStockBrands, error)
}

type getHighVolumeStockBrandsInteractor struct {
	highVolumeStockBrandRepo repositories.HighVolumeStockBrandRepository
}

func NewGetHighVolumeStockBrandsUseCase(
	highVolumeStockBrandRepo repositories.HighVolumeStockBrandRepository,
) GetHighVolumeStockBrandsUseCase {
	return &getHighVolumeStockBrandsInteractor{
		highVolumeStockBrandRepo: highVolumeStockBrandRepo,
	}
}

func (u *getHighVolumeStockBrandsInteractor) Execute(ctx context.Context) ([]*models.HighVolumeStockBrand, error) {
	brands, err := u.highVolumeStockBrandRepo.FindAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve high volume stock brands")
	}

	return brands, nil
}

func (u *getHighVolumeStockBrandsInteractor) ExecuteWithPagination(
	ctx context.Context,
	symbolFrom string,
	limit int,
) (*models.PaginatedHighVolumeStockBrands, error) {
	// Validate limit parameter
	if limit < 0 {
		return nil, errors.New("limit must be non-negative")
	}

	// Fetch limit+1 to determine if there are more pages
	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	brands, err := u.highVolumeStockBrandRepo.FindWithPagination(ctx, symbolFrom, fetchLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve paginated high volume stock brands")
	}

	result := &models.PaginatedHighVolumeStockBrands{
		Brands: brands,
		Limit:  limit,
	}

	// Set next cursor if there are more results
	if limit > 0 && len(brands) > limit {
		// NextCursorは現在のページの最後の要素のTickerSymbol
		lastBrand := brands[limit-1]
		result.NextCursor = &lastBrand.TickerSymbol
		result.Brands = brands[:limit] // Trim to requested limit
	}

	return result, nil
}

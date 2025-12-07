//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

// HighVolumeStockBrandWithName is a custom struct for JOIN query results
type HighVolumeStockBrandWithName struct {
	genModel.HighVolumeStockBrand
	CompanyName string `gorm:"column:name"`
}

// HighVolumeStockBrandRepositoryImpl implements HighVolumeStockBrandRepository
type HighVolumeStockBrandRepositoryImpl struct {
	query *genQuery.Query
}

func NewHighVolumeStockBrandRepositoryImpl(db *gorm.DB) repositories.HighVolumeStockBrandRepository {
	return &HighVolumeStockBrandRepositoryImpl{
		query: genQuery.Use(db),
	}
}

// FindAll retrieves all high volume stock brands from the database
func (r *HighVolumeStockBrandRepositoryImpl) FindAll(ctx context.Context) ([]*models.HighVolumeStockBrand, error) {
	return r.FindWithPagination(ctx, "", 0)
}

// FindWithPagination retrieves high volume stock brands with cursor-based pagination
func (r *HighVolumeStockBrandRepositoryImpl) FindWithPagination(
	ctx context.Context,
	symbolFrom string,
	limit int,
) ([]*models.HighVolumeStockBrand, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	// Get the underlying *gorm.DB from query
	db := tx.HighVolumeStockBrand.UnderlyingDB()

	// Build JOIN query using raw GORM
	query := db.WithContext(ctx).Table("high_volume_stock_brands hvb").
		Select("hvb.stock_brand_id, hvb.ticker_symbol, hvb.volume_average, hvb.created_at, sb.name").
		Joins("LEFT JOIN stock_brand sb ON hvb.stock_brand_id = sb.id")

	// Apply cursor condition
	if symbolFrom != "" {
		query = query.Where("hvb.ticker_symbol > ?", symbolFrom)
	}

	// Apply limit
	if limit > 0 {
		query = query.Limit(limit)
	}

	// Order and execute
	var resultRows []HighVolumeStockBrandWithName
	if err := query.Order("hvb.ticker_symbol").Find(&resultRows).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(err, "HighVolumeStockBrandRepositoryImpl.FindWithPagination error")
		}
		return []*models.HighVolumeStockBrand{}, nil
	}

	if len(resultRows) == 0 {
		return []*models.HighVolumeStockBrand{}, nil
	}

	result := make([]*models.HighVolumeStockBrand, 0, len(resultRows))
	for _, v := range resultRows {
		result = append(result, r.convertToDomainModelWithName(&v.HighVolumeStockBrand, v.CompanyName))
	}
	return result, nil
}

func (r *HighVolumeStockBrandRepositoryImpl) convertToDomainModelWithName(
	dbModel *genModel.HighVolumeStockBrand,
	companyName string,
) *models.HighVolumeStockBrand {
	return models.NewHighVolumeStockBrand(
		dbModel.StockBrandID,
		dbModel.TickerSymbol,
		companyName,
		dbModel.VolumeAverage,
		dbModel.CreatedAt,
	)
}

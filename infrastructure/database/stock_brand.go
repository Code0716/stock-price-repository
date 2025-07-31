//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	stockBrandMarketCodePrime    string = "111"
	stockBrandMarketCodeStandard string = "112"
	stockBrandMarketCodeGrowth   string = "113"
)

// StockBrandRepositoryImpl implements  StockBrandRepository
type StockBrandRepositoryImpl struct {
	query *genQuery.Query
}

func NewStockBrandRepositoryImpl(db *gorm.DB) repositories.StockBrandRepository {
	return &StockBrandRepositoryImpl{
		query: genQuery.Use(db),
	}
}

// FindAll retrieves all stock brands from the database.
func (si *StockBrandRepositoryImpl) FindAll(ctx context.Context) ([]*models.StockBrand, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	resultRow, err := tx.StockBrand.WithContext(ctx).
		Where(tx.StockBrand.DeletedAt.IsNull()).
		Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "StockBrandRepositoryImpl.FindAll error")
	}
	if err != nil {
		return nil, nil
	}

	result := make([]*models.StockBrand, 0, len(resultRow))
	for _, v := range resultRow {
		result = append(result, si.convertToDomainModel(v))
	}
	return result, nil
}

func (si *StockBrandRepositoryImpl) UpsertStockBrands(ctx context.Context, stockBrands []*models.StockBrand) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	err := tx.StockBrand.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			Where: clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: "deleted_at", Value: nil},
			}},
			DoUpdates: clause.AssignmentColumns(
				[]string{
					"name",
					"market_code",
					"market_name",
					"sector_33_code",
					"sector_33_code_name",
					"sector_17_code",
					"sector_17_code_name",
					"updated_at",
				}),
		}).Create(si.convertToDBModels(stockBrands)...)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (si *StockBrandRepositoryImpl) FindFromSymbol(ctx context.Context, symbol string, limit int) ([]*models.StockBrand, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	resultRow, err := tx.StockBrand.Where(
		tx.StockBrand.TickerSymbol.Gt(symbol),
	).Where(
		tx.StockBrand.MarketCode.In(stockBrandMarketCodePrime, stockBrandMarketCodeStandard, stockBrandMarketCodeGrowth),
	).Limit(limit).
		Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "StockBrandRepositoryImpl.FindBySymbol error")
	}
	if err != nil {
		return nil, nil
	}

	result := make([]*models.StockBrand, 0, len(resultRow))
	for _, v := range resultRow {
		result = append(result, si.convertToDomainModel(v))
	}
	return result, nil
}

func (si *StockBrandRepositoryImpl) FindDelistingStockBrandsFromUpdateTime(ctx context.Context, now time.Time) ([]string, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	resultRow, err := tx.StockBrand.Where(tx.StockBrand.UpdatedAt.Lt(now)).Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "StockBrandRepositoryImpl.FindDelistingStockBrandsFromUpdateTime error")
	}

	ids := make([]string, 0, len(resultRow))
	for _, v := range resultRow {
		ids = append(ids, v.ID)
	}

	return ids, nil
}

func (si *StockBrandRepositoryImpl) DeleteDelistingStockBrands(ctx context.Context, ids []string) ([]*models.StockBrand, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = si.query
	}

	if len(ids) == 0 {
		return nil, nil
	}

	deleteBrands, err := tx.StockBrand.WithContext(ctx).
		Where(tx.StockBrand.ID.In(ids...)).
		Where(tx.StockBrand.DeletedAt.IsNull()).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "StockBrandRepositoryImpl.DeleteDelistingStockBrands error")
	}

	if _, err := tx.StockBrand.WithContext(ctx).
		Where(tx.StockBrand.ID.In(ids...)).
		Delete(); err != nil {
		return nil, errors.Wrap(err, "StockBrandRepositoryImpl.DeleteDelistingStockBrands error")
	}

	results := make([]*models.StockBrand, 0, len(deleteBrands))
	for _, v := range deleteBrands {
		results = append(results, si.convertToDomainModel(v))

	}
	return results, nil
}

func (si *StockBrandRepositoryImpl) convertToDBModels(stockBrands []*models.StockBrand) []*genModel.StockBrand {
	var stockBrandsDB []*genModel.StockBrand
	for _, v := range stockBrands {
		stockBrandsDB = append(stockBrandsDB, si.convertToDBModel(v))
	}
	return stockBrandsDB
}

func (si *StockBrandRepositoryImpl) convertToDBModel(stockBrand *models.StockBrand) *genModel.StockBrand {
	return &genModel.StockBrand{
		ID:               stockBrand.ID,
		TickerSymbol:     stockBrand.TickerSymbol,
		Name:             stockBrand.Name,
		MarketCode:       stockBrand.MarketCode,
		MarketName:       stockBrand.MarketName,
		Sector33Code:     &stockBrand.Sector33Code,
		Sector33CodeName: &stockBrand.Sector33CodeName,
		Sector17Code:     &stockBrand.Sector17Code,
		Sector17CodeName: &stockBrand.Sector17CodeName,
		CreatedAt:        stockBrand.CreatedAt,
		UpdatedAt:        stockBrand.UpdatedAt,
	}
}

func (si *StockBrandRepositoryImpl) convertToDomainModel(stockBrand *genModel.StockBrand) *models.StockBrand {
	return &models.StockBrand{
		ID:               stockBrand.ID,
		TickerSymbol:     stockBrand.TickerSymbol,
		Name:             stockBrand.Name,
		MarketCode:       stockBrand.MarketCode,
		MarketName:       stockBrand.MarketName,
		Sector33Code:     *stockBrand.Sector33Code,
		Sector33CodeName: *stockBrand.Sector33CodeName,
		Sector17Code:     *stockBrand.Sector17Code,
		Sector17CodeName: *stockBrand.Sector17CodeName,
		CreatedAt:        stockBrand.CreatedAt,
		UpdatedAt:        stockBrand.UpdatedAt,
	}
}

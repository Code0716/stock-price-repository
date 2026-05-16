//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type FinAnnouncementRepositoryImpl struct {
	db *gorm.DB
}

func NewFinAnnouncementRepositoryImpl(db *gorm.DB) repositories.FinAnnouncementRepository {
	return &FinAnnouncementRepositoryImpl{db: db}
}

type finAnnouncementRow struct {
	ID               string    `gorm:"column:id"`
	TickerSymbol     string    `gorm:"column:ticker_symbol"`
	StockBrandID     *string   `gorm:"column:stock_brand_id"`
	AnnouncementDate time.Time `gorm:"column:announcement_date"`
	FiscalYear       string    `gorm:"column:fiscal_year"`
	FiscalQuarter    string    `gorm:"column:fiscal_quarter"`
	Sector17Code     string    `gorm:"column:sector_17_code"`
	Sector33Code     string    `gorm:"column:sector_33_code"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (finAnnouncementRow) TableName() string { return "fin_announcement" }

func (r *FinAnnouncementRepositoryImpl) Upsert(ctx context.Context, announcements []*models.FinAnnouncement) error {
	if len(announcements) == 0 {
		return nil
	}

	rows := make([]*finAnnouncementRow, 0, len(announcements))
	for _, a := range announcements {
		rows = append(rows, &finAnnouncementRow{
			ID:               a.ID,
			TickerSymbol:     a.TickerSymbol,
			StockBrandID:     a.StockBrandID,
			AnnouncementDate: a.AnnouncementDate,
			FiscalYear:       a.FiscalYear,
			FiscalQuarter:    a.FiscalQuarter,
			Sector17Code:     a.Sector17Code,
			Sector33Code:     a.Sector33Code,
			CreatedAt:        a.CreatedAt,
			UpdatedAt:        a.UpdatedAt,
		})
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ticker_symbol"}, {Name: "announcement_date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"fiscal_year", "fiscal_quarter", "sector_17_code", "sector_33_code", "updated_at",
			}),
		}).
		Create(&rows).Error; err != nil {
		return errors.Wrap(err, "FinAnnouncementRepositoryImpl.Upsert error")
	}
	return nil
}

func (r *FinAnnouncementRepositoryImpl) FindWithFilter(ctx context.Context, filter *models.FinAnnouncementFilter) ([]*models.FinAnnouncement, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	db := r.buildQuery(ctx, filter)
	offset := (filter.Page - 1) * filter.Limit

	var rows []*finAnnouncementRow
	if err := db.Order("announcement_date ASC").Limit(filter.Limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "FinAnnouncementRepositoryImpl.FindWithFilter error")
	}

	result := make([]*models.FinAnnouncement, 0, len(rows))
	for _, row := range rows {
		result = append(result, r.convertToDomainModel(row))
	}
	return result, nil
}

func (r *FinAnnouncementRepositoryImpl) CountWithFilter(ctx context.Context, filter *models.FinAnnouncementFilter) (int64, error) {
	db := r.buildQuery(ctx, filter)
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, "FinAnnouncementRepositoryImpl.CountWithFilter error")
	}
	return count, nil
}

func (r *FinAnnouncementRepositoryImpl) FindNextBySymbol(ctx context.Context, tickerSymbol string) (*models.FinAnnouncement, error) {
	var row finAnnouncementRow
	err := r.db.WithContext(ctx).
		Table("fin_announcement").
		Where("ticker_symbol = ? AND announcement_date >= ?", tickerSymbol, time.Now().Truncate(24*time.Hour)).
		Order("announcement_date ASC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "FinAnnouncementRepositoryImpl.FindNextBySymbol error")
	}
	return r.convertToDomainModel(&row), nil
}

func (r *FinAnnouncementRepositoryImpl) buildQuery(ctx context.Context, filter *models.FinAnnouncementFilter) *gorm.DB {
	db := r.db.WithContext(ctx).Table("fin_announcement")
	if filter.TickerSymbol != "" {
		db = db.Where("ticker_symbol = ?", filter.TickerSymbol)
	}
	if !filter.From.IsZero() {
		db = db.Where("announcement_date >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		db = db.Where("announcement_date <= ?", filter.To)
	}
	return db
}

func (r *FinAnnouncementRepositoryImpl) convertToDomainModel(row *finAnnouncementRow) *models.FinAnnouncement {
	return &models.FinAnnouncement{
		ID:               row.ID,
		TickerSymbol:     row.TickerSymbol,
		StockBrandID:     row.StockBrandID,
		AnnouncementDate: row.AnnouncementDate,
		FiscalYear:       row.FiscalYear,
		FiscalQuarter:    row.FiscalQuarter,
		Sector17Code:     row.Sector17Code,
		Sector33Code:     row.Sector33Code,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}


//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type QuizDailyUniverseRepositoryImpl struct {
	query *genQuery.Query
}

func NewQuizDailyUniverseRepositoryImpl(db *gorm.DB) repositories.QuizDailyUniverseRepository {
	return &QuizDailyUniverseRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (qi *QuizDailyUniverseRepositoryImpl) BulkCreate(ctx context.Context, entries []*models.QuizUniverseEntry) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	if len(entries) == 0 {
		return nil
	}

	if err := tx.QuizDailyUniverse.WithContext(ctx).
		Create(qi.convertToDBModels(entries)...); err != nil {
		return errors.Wrap(err, "QuizDailyUniverseRepositoryImpl.BulkCreate error")
	}
	return nil
}

func (qi *QuizDailyUniverseRepositoryImpl) ListByQuizDate(ctx context.Context, quizDate time.Time) ([]*models.QuizUniverseEntry, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	dateOnly := dateOnlyOf(quizDate)

	rows, err := tx.QuizDailyUniverse.WithContext(ctx).
		Where(tx.QuizDailyUniverse.QuizDate.Eq(dateOnly)).
		Order(tx.QuizDailyUniverse.QuestionOrder).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "QuizDailyUniverseRepositoryImpl.ListByQuizDate error")
	}

	entries := make([]*models.QuizUniverseEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, qi.convertToDomainModel(r))
	}
	return entries, nil
}

func (qi *QuizDailyUniverseRepositoryImpl) FindLatestQuizDate(ctx context.Context) (*time.Time, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	row, err := tx.QuizDailyUniverse.WithContext(ctx).
		Order(tx.QuizDailyUniverse.QuizDate.Desc()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "QuizDailyUniverseRepositoryImpl.FindLatestQuizDate error")
	}

	return &row.QuizDate, nil
}

func (qi *QuizDailyUniverseRepositoryImpl) FindByQuizDateAndStockBrandID(ctx context.Context, quizDate time.Time, stockBrandID string) (*models.QuizUniverseEntry, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	row, err := tx.QuizDailyUniverse.WithContext(ctx).
		Where(tx.QuizDailyUniverse.QuizDate.Eq(dateOnlyOf(quizDate))).
		Where(tx.QuizDailyUniverse.StockBrandID.Eq(stockBrandID)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "QuizDailyUniverseRepositoryImpl.FindByQuizDateAndStockBrandID error")
	}

	return qi.convertToDomainModel(row), nil
}

func (qi *QuizDailyUniverseRepositoryImpl) ExistsByQuizDate(ctx context.Context, quizDate time.Time) (bool, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	count, err := tx.QuizDailyUniverse.WithContext(ctx).
		Where(tx.QuizDailyUniverse.QuizDate.Eq(dateOnlyOf(quizDate))).
		Count()
	if err != nil {
		return false, errors.Wrap(err, "QuizDailyUniverseRepositoryImpl.ExistsByQuizDate error")
	}

	return count > 0, nil
}

func dateOnlyOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (qi *QuizDailyUniverseRepositoryImpl) convertToDomainModel(m *genModel.QuizDailyUniverse) *models.QuizUniverseEntry {
	return &models.QuizUniverseEntry{
		QuizDate:        m.QuizDate,
		StockBrandID:    m.StockBrandID,
		TickerSymbol:    m.TickerSymbol,
		QuestionOrder:   int(m.QuestionOrder),
		AvgTradingValue: decimal.NewFromFloat(m.AvgTradingValue),
		AvgDailyRange:   decimal.NewFromFloat(m.AvgDailyRange),
		BaseClosePrice:  decimal.NewFromFloat(m.BaseClosePrice),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func (qi *QuizDailyUniverseRepositoryImpl) convertToDBModels(entries []*models.QuizUniverseEntry) []*genModel.QuizDailyUniverse {
	dbEntries := make([]*genModel.QuizDailyUniverse, 0, len(entries))
	for _, e := range entries {
		avgTradingValue, _ := e.AvgTradingValue.Round(4).Float64()
		avgDailyRange, _ := e.AvgDailyRange.Round(6).Float64()
		baseClosePrice, _ := e.BaseClosePrice.Round(4).Float64()
		dbEntries = append(dbEntries, &genModel.QuizDailyUniverse{
			QuizDate:        dateOnlyOf(e.QuizDate),
			StockBrandID:    e.StockBrandID,
			TickerSymbol:    e.TickerSymbol,
			QuestionOrder:   uint32(e.QuestionOrder),
			AvgTradingValue: avgTradingValue,
			AvgDailyRange:   avgDailyRange,
			BaseClosePrice:  baseClosePrice,
		})
	}
	return dbEntries
}

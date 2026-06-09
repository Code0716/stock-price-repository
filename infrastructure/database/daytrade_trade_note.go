package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type DaytradeTradeNoteRepositoryImpl struct {
	query *genQuery.Query
	db    *gorm.DB
}

func NewDaytradeTradeNoteRepositoryImpl(db *gorm.DB) repositories.DaytradeTradeNoteRepository {
	return &DaytradeTradeNoteRepositoryImpl{
		query: genQuery.Use(db),
		db:    db,
	}
}

func (r *DaytradeTradeNoteRepositoryImpl) FindByDateRange(ctx context.Context, from, to *time.Time) ([]*models.DaytradeTradeNoteRecord, error) {
	q := r.query.DaytradeTradeNote
	tx, ok := GetTxQuery(ctx)
	if ok {
		q = tx.DaytradeTradeNote
	}

	stmt := q.WithContext(ctx)
	if from != nil {
		stmt = stmt.Where(q.ExecutedOn.Gte(*from))
	}
	if to != nil {
		stmt = stmt.Where(q.ExecutedOn.Lte(*to))
	}

	rows, err := stmt.Order(q.ExecutedOn.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, errors.Wrap(err, "DaytradeTradeNoteRepositoryImpl.FindByDateRange error")
	}

	result := make([]*models.DaytradeTradeNoteRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := r.convertToDomainModel(row)
		if err != nil {
			return nil, errors.Wrap(err, "convertToDomainModel error")
		}
		result = append(result, rec)
	}
	return result, nil
}

func (r *DaytradeTradeNoteRepositoryImpl) Upsert(ctx context.Context, rec *models.DaytradeTradeNoteRecord) error {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	// 全フィールド空なら行削除
	if rec.Memo == "" && len(rec.Tags) == 0 && rec.DeclaredStopPrice == nil {
		result := db.WithContext(ctx).
			Where("ticker_symbol = ? AND executed_on = ? AND direction = ?",
				rec.TickerSymbol, rec.ExecutedOn, rec.Direction).
			Delete(&genModel.DaytradeTradeNote{})
		if result.Error != nil {
			return errors.Wrap(result.Error, "DaytradeTradeNoteRepositoryImpl.Upsert (delete) error")
		}
		return nil
	}

	row, err := r.convertToDBModel(rec)
	if err != nil {
		return errors.Wrap(err, "convertToDBModel error")
	}

	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "executed_on"},
				{Name: "ticker_symbol"},
				{Name: "direction"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"memo", "tags", "declared_stop_price", "updated_at"}),
		}).
		Create(row)
	if result.Error != nil {
		return errors.Wrap(result.Error, "DaytradeTradeNoteRepositoryImpl.Upsert error")
	}
	return nil
}

func (r *DaytradeTradeNoteRepositoryImpl) convertToDomainModel(row *genModel.DaytradeTradeNote) (*models.DaytradeTradeNoteRecord, error) {
	rec := &models.DaytradeTradeNoteRecord{
		TickerSymbol: row.TickerSymbol,
		ExecutedOn:   row.ExecutedOn,
		Direction:    row.Direction,
	}
	if row.Memo != nil {
		rec.Memo = *row.Memo
	}
	if row.Tags != nil {
		var tags []string
		if err := json.Unmarshal([]byte(*row.Tags), &tags); err != nil {
			return nil, errors.Wrap(err, "json.Unmarshal tags error")
		}
		rec.Tags = tags
	}
	if row.DeclaredStopPrice != nil {
		d := decimal.NewFromFloat(*row.DeclaredStopPrice)
		rec.DeclaredStopPrice = &d
	}
	return rec, nil
}

func (r *DaytradeTradeNoteRepositoryImpl) convertToDBModel(rec *models.DaytradeTradeNoteRecord) (*genModel.DaytradeTradeNote, error) {
	row := &genModel.DaytradeTradeNote{
		TickerSymbol: rec.TickerSymbol,
		ExecutedOn:   rec.ExecutedOn,
		Direction:    rec.Direction,
	}
	if rec.Memo != "" {
		row.Memo = &rec.Memo
	}
	if len(rec.Tags) > 0 {
		b, err := json.Marshal(rec.Tags)
		if err != nil {
			return nil, errors.Wrap(err, "json.Marshal tags error")
		}
		s := string(b)
		row.Tags = &s
	}
	if rec.DeclaredStopPrice != nil {
		f, _ := rec.DeclaredStopPrice.Float64()
		row.DeclaredStopPrice = &f
	}
	return row, nil
}

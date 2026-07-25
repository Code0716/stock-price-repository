//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type DailyStockPickRepositoryImpl struct {
	query *genQuery.Query
}

func NewDailyStockPickRepositoryImpl(db *gorm.DB) repositories.DailyStockPickRepository {
	return &DailyStockPickRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (di *DailyStockPickRepositoryImpl) BulkCreate(ctx context.Context, picks []*models.DailyStockPick) error {
	tx := TxOrDefault(ctx, di.query)

	if len(picks) == 0 {
		return nil
	}

	if err := tx.DailyStockPick.WithContext(ctx).
		Create(di.convertToDBModels(picks)...); err != nil {
		return errors.Wrap(err, "DailyStockPickRepositoryImpl.BulkCreate error")
	}
	return nil
}

func (di *DailyStockPickRepositoryImpl) DeleteByPickDate(ctx context.Context, pickDate time.Time) error {
	tx := TxOrDefault(ctx, di.query)

	if _, err := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.PickDate.Eq(dateOnlyOf(pickDate))).
		Delete(); err != nil {
		return errors.Wrap(err, "DailyStockPickRepositoryImpl.DeleteByPickDate error")
	}
	return nil
}

func (di *DailyStockPickRepositoryImpl) ListByPickDate(ctx context.Context, pickDate time.Time) ([]*models.DailyStockPick, error) {
	tx := TxOrDefault(ctx, di.query)

	rows, err := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.PickDate.Eq(dateOnlyOf(pickDate))).
		Order(tx.DailyStockPick.PickRank).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "DailyStockPickRepositoryImpl.ListByPickDate error")
	}

	picks := make([]*models.DailyStockPick, 0, len(rows))
	for _, r := range rows {
		picks = append(picks, di.convertToDomainModel(r))
	}
	return picks, nil
}

func (di *DailyStockPickRepositoryImpl) ExistsByPickDate(ctx context.Context, pickDate time.Time) (bool, error) {
	tx := TxOrDefault(ctx, di.query)

	count, err := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.PickDate.Eq(dateOnlyOf(pickDate))).
		Count()
	if err != nil {
		return false, errors.Wrap(err, "DailyStockPickRepositoryImpl.ExistsByPickDate error")
	}
	return count > 0, nil
}

func (di *DailyStockPickRepositoryImpl) ListPendingEvaluation(ctx context.Context, onOrAfter time.Time) ([]*models.DailyStockPick, error) {
	tx := TxOrDefault(ctx, di.query)

	rows, err := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.EvaluatedAt.IsNull()).
		Where(tx.DailyStockPick.PickDate.Gte(dateOnlyOf(onOrAfter))).
		Order(tx.DailyStockPick.PickDate).
		Find()
	if err != nil {
		return nil, errors.Wrap(err, "DailyStockPickRepositoryImpl.ListPendingEvaluation error")
	}

	picks := make([]*models.DailyStockPick, 0, len(rows))
	for _, r := range rows {
		picks = append(picks, di.convertToDomainModel(r))
	}
	return picks, nil
}

func (di *DailyStockPickRepositoryImpl) UpdateEvaluations(ctx context.Context, picks []*models.DailyStockPick) error {
	tx := TxOrDefault(ctx, di.query)

	for _, p := range picks {
		if _, err := tx.DailyStockPick.WithContext(ctx).
			Where(tx.DailyStockPick.PickDate.Eq(dateOnlyOf(p.PickDate))).
			Where(tx.DailyStockPick.StockBrandID.Eq(p.StockBrandID)).
			Updates(di.convertToDBModel(p)); err != nil {
			return errors.Wrap(err, "DailyStockPickRepositoryImpl.UpdateEvaluations error")
		}
	}
	return nil
}

func (di *DailyStockPickRepositoryImpl) MarkNotified(ctx context.Context, pickDate time.Time, notifiedAt time.Time) error {
	tx := TxOrDefault(ctx, di.query)

	if _, err := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.PickDate.Eq(dateOnlyOf(pickDate))).
		Update(tx.DailyStockPick.NotifiedAt, notifiedAt); err != nil {
		return errors.Wrap(err, "DailyStockPickRepositoryImpl.MarkNotified error")
	}
	return nil
}

func (di *DailyStockPickRepositoryImpl) FindLatestPickDate(ctx context.Context) (*time.Time, error) {
	tx := TxOrDefault(ctx, di.query)

	row, err := tx.DailyStockPick.WithContext(ctx).
		Order(tx.DailyStockPick.PickDate.Desc()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "DailyStockPickRepositoryImpl.FindLatestPickDate error")
	}

	return &row.PickDate, nil
}

func (di *DailyStockPickRepositoryImpl) ListPickDates(ctx context.Context, limit int) ([]time.Time, error) {
	tx := TxOrDefault(ctx, di.query)

	q := tx.DailyStockPick.WithContext(ctx).
		Distinct(tx.DailyStockPick.PickDate).
		Order(tx.DailyStockPick.PickDate.Desc())
	if limit > 0 {
		q = q.Limit(limit)
	}

	var dates []time.Time
	if err := q.Pluck(tx.DailyStockPick.PickDate, &dates); err != nil {
		return nil, errors.Wrap(err, "DailyStockPickRepositoryImpl.ListPickDates error")
	}
	return dates, nil
}

func (di *DailyStockPickRepositoryImpl) ListByDateRange(ctx context.Context, from, to *time.Time, scoreVersion string) ([]*models.DailyStockPick, error) {
	tx := TxOrDefault(ctx, di.query)

	q := tx.DailyStockPick.WithContext(ctx).
		Where(tx.DailyStockPick.ScoreVersion.Eq(scoreVersion))
	if from != nil {
		q = q.Where(tx.DailyStockPick.PickDate.Gte(dateOnlyOf(*from)))
	}
	if to != nil {
		q = q.Where(tx.DailyStockPick.PickDate.Lte(dateOnlyOf(*to)))
	}

	rows, err := q.Order(tx.DailyStockPick.PickDate, tx.DailyStockPick.PickRank).Find()
	if err != nil {
		return nil, errors.Wrap(err, "DailyStockPickRepositoryImpl.ListByDateRange error")
	}

	picks := make([]*models.DailyStockPick, 0, len(rows))
	for _, r := range rows {
		picks = append(picks, di.convertToDomainModel(r))
	}
	return picks, nil
}

func (di *DailyStockPickRepositoryImpl) convertToDomainModel(m *genModel.DailyStockPick) *models.DailyStockPick {
	p := &models.DailyStockPick{
		PickDate:          m.PickDate,
		StockBrandID:      m.StockBrandID,
		TickerSymbol:      m.TickerSymbol,
		PickRank:          int(m.PickRank),
		Score:             decimal.NewFromFloat(m.Score),
		ScoreVersion:      m.ScoreVersion,
		SignalCount:       int(m.SignalCount),
		Strategies:        splitStrategies(m.Strategies),
		AvgTradingValue:   decimal.NewFromFloat(m.AvgTradingValue),
		BaseClosePrice:    decimal.NewFromFloat(m.BaseClosePrice),
		BaseAdjClosePrice: decimal.NewFromFloat(m.BaseAdjClosePrice),
		VolumeRatio:       decimal.NewFromFloat(m.VolumeRatio),
		ADX:               decimal.NewFromFloat(m.Adx),
		PlusDI:            decimal.NewFromFloat(m.PlusDi),
		MinusDI:           decimal.NewFromFloat(m.MinusDi),
		ATRRatio:          decimal.NewFromFloat(m.AtrRatio),
		RSI:               decimal.NewFromFloat(m.Rsi),
		NotifiedAt:        m.NotifiedAt,
		EvaluatedAt:       m.EvaluatedAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
	if m.Sector33CodeName != nil {
		p.Sector33CodeName = *m.Sector33CodeName
	}
	if m.Return1D != nil {
		v := decimal.NewFromFloat(*m.Return1D)
		p.Return1D = &v
	}
	if m.Return3D != nil {
		v := decimal.NewFromFloat(*m.Return3D)
		p.Return3D = &v
	}
	if m.Return5D != nil {
		v := decimal.NewFromFloat(*m.Return5D)
		p.Return5D = &v
	}
	if m.Outcome != nil {
		v := models.DailyStockPickOutcome(*m.Outcome)
		p.Outcome = &v
	}
	return p
}

func (di *DailyStockPickRepositoryImpl) convertToDBModel(p *models.DailyStockPick) *genModel.DailyStockPick {
	m := &genModel.DailyStockPick{
		PickDate:          dateOnlyOf(p.PickDate),
		StockBrandID:      p.StockBrandID,
		TickerSymbol:      p.TickerSymbol,
		PickRank:          uint32(p.PickRank),
		Score:             roundToFloat64(p.Score, 2),
		ScoreVersion:      p.ScoreVersion,
		SignalCount:       uint32(p.SignalCount),
		Strategies:        strings.Join(p.Strategies, ","),
		BaseClosePrice:    roundToFloat64(p.BaseClosePrice, 4),
		BaseAdjClosePrice: roundToFloat64(p.BaseAdjClosePrice, 4),
		AvgTradingValue:   roundToFloat64(p.AvgTradingValue, 4),
		VolumeRatio:       roundToFloat64(p.VolumeRatio, 4),
		Adx:               roundToFloat64(p.ADX, 4),
		PlusDi:            roundToFloat64(p.PlusDI, 4),
		MinusDi:           roundToFloat64(p.MinusDI, 4),
		AtrRatio:          roundToFloat64(p.ATRRatio, 6),
		Rsi:               roundToFloat64(p.RSI, 4),
		NotifiedAt:        p.NotifiedAt,
		EvaluatedAt:       p.EvaluatedAt,
	}
	if p.Sector33CodeName != "" {
		v := p.Sector33CodeName
		m.Sector33CodeName = &v
	}
	if p.Return1D != nil {
		v := roundToFloat64(*p.Return1D, 6)
		m.Return1D = &v
	}
	if p.Return3D != nil {
		v := roundToFloat64(*p.Return3D, 6)
		m.Return3D = &v
	}
	if p.Return5D != nil {
		v := roundToFloat64(*p.Return5D, 6)
		m.Return5D = &v
	}
	if p.Outcome != nil {
		v := string(*p.Outcome)
		m.Outcome = &v
	}
	return m
}

func (di *DailyStockPickRepositoryImpl) convertToDBModels(picks []*models.DailyStockPick) []*genModel.DailyStockPick {
	out := make([]*genModel.DailyStockPick, 0, len(picks))
	for _, p := range picks {
		out = append(out, di.convertToDBModel(p))
	}
	return out
}

func roundToFloat64(d decimal.Decimal, places int32) float64 {
	v, _ := d.Round(places).Float64()
	return v
}

func splitStrategies(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

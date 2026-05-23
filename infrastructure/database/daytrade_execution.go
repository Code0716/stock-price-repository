package database

import (
	"context"
	"database/sql"
	"fmt"
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

type DaytradeExecutionRepositoryImpl struct {
	query *genQuery.Query
	db    *gorm.DB
}

func NewDaytradeExecutionRepositoryImpl(db *gorm.DB) repositories.DaytradeExecutionRepository {
	return &DaytradeExecutionRepositoryImpl{
		query: genQuery.Use(db),
		db:    db,
	}
}

func (r *DaytradeExecutionRepositoryImpl) DeleteBySource(ctx context.Context, source string) (int64, error) {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	result := db.WithContext(ctx).
		Where("source = ?", source).
		Delete(&genModel.DaytradeExecution{})
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, "DaytradeExecutionRepositoryImpl.DeleteBySource error")
	}
	return result.RowsAffected, nil
}

func (r *DaytradeExecutionRepositoryImpl) BulkInsertIgnore(ctx context.Context, executions []*models.DaytradeExecution) (int, error) {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	rows := r.convertToDBModels(executions)
	result := db.WithContext(ctx).
		Clauses(clause.Insert{Modifier: "IGNORE"}).
		CreateInBatches(rows, 500)
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, "DaytradeExecutionRepositoryImpl.BulkInsertIgnore error")
	}
	return int(result.RowsAffected), nil
}

type aggregateRow struct {
	BucketDate  sql.NullString `gorm:"column:bucket_date"`
	ProfitLoss  int64          `gorm:"column:profit_loss"`
	TradeCount  int            `gorm:"column:trade_count"`
	GrossProfit int64          `gorm:"column:gross_profit"`
	GrossLoss   int64          `gorm:"column:gross_loss"`
	WinCount    int            `gorm:"column:win_count"`
	LossCount   int            `gorm:"column:loss_count"`
}

func (r *DaytradeExecutionRepositoryImpl) Aggregate(ctx context.Context, from, to *time.Time, g models.DaytradeSummaryGranularity) ([]*models.DaytradeSummaryBucket, error) {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	var selectExpr, groupByExpr string
	switch g {
	case models.DaytradeSummaryGranularityDaily:
		selectExpr = "DATE_FORMAT(executed_on, '%Y-%m-%d') AS bucket_date"
		groupByExpr = "bucket_date"
	case models.DaytradeSummaryGranularityMonthly:
		selectExpr = "DATE_FORMAT(executed_on, '%Y-%m-01') AS bucket_date"
		groupByExpr = "bucket_date"
	case models.DaytradeSummaryGranularityYearly:
		selectExpr = "DATE_FORMAT(executed_on, '%Y-01-01') AS bucket_date"
		groupByExpr = "bucket_date"
	case models.DaytradeSummaryGranularityAll:
		selectExpr = "NULL AS bucket_date"
		groupByExpr = ""
	default:
		return nil, errors.Errorf("unknown granularity: %s", g)
	}

	query := db.WithContext(ctx).
		Table("daytrade_executions").
		Select(fmt.Sprintf(
			"%s,"+
				" SUM(profit_loss) AS profit_loss,"+
				" COUNT(*) AS trade_count,"+
				" COALESCE(SUM(CASE WHEN profit_loss > 0 THEN profit_loss ELSE 0 END), 0) AS gross_profit,"+
				" COALESCE(SUM(CASE WHEN profit_loss < 0 THEN profit_loss ELSE 0 END), 0) AS gross_loss,"+
				" COALESCE(SUM(CASE WHEN profit_loss > 0 THEN 1 ELSE 0 END), 0) AS win_count,"+
				" COALESCE(SUM(CASE WHEN profit_loss < 0 THEN 1 ELSE 0 END), 0) AS loss_count",
			selectExpr,
		))

	if from != nil {
		query = query.Where("executed_on >= ?", from.Format("2006-01-02"))
	}
	if to != nil {
		query = query.Where("executed_on <= ?", to.Format("2006-01-02"))
	}

	if groupByExpr != "" {
		query = query.Group(groupByExpr).Order("bucket_date ASC")
	}

	var rows []aggregateRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "DaytradeExecutionRepositoryImpl.Aggregate error")
	}

	buckets := make([]*models.DaytradeSummaryBucket, 0, len(rows))
	for _, row := range rows {
		bucket := &models.DaytradeSummaryBucket{
			ProfitLoss:  row.ProfitLoss,
			TradeCount:  row.TradeCount,
			GrossProfit: row.GrossProfit,
			GrossLoss:   row.GrossLoss,
			WinCount:    row.WinCount,
			LossCount:   row.LossCount,
		}
		if row.BucketDate.Valid {
			s := row.BucketDate.String
			bucket.BucketDate = &s
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

type symbolSummaryRow struct {
	TickerSymbol string `gorm:"column:ticker_symbol"`
	BrandName    string `gorm:"column:brand_name"`
	ProfitLoss   int64  `gorm:"column:profit_loss"`
	TradeCount   int    `gorm:"column:trade_count"`
	GrossProfit  int64  `gorm:"column:gross_profit"`
	GrossLoss    int64  `gorm:"column:gross_loss"`
	WinCount     int    `gorm:"column:win_count"`
	LossCount    int    `gorm:"column:loss_count"`
}

func (r *DaytradeExecutionRepositoryImpl) AggregateByTickerSymbol(ctx context.Context, from, to *time.Time) ([]*models.DaytradeSymbolSummary, error) {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	query := db.WithContext(ctx).
		Table("daytrade_executions AS d").
		Joins("LEFT JOIN stock_brand AS b ON d.ticker_symbol = b.ticker_symbol AND b.deleted_at IS NULL").
		Select(
			"d.ticker_symbol," +
				" COALESCE(MAX(b.name), MAX(d.brand_name)) AS brand_name," +
				" SUM(d.profit_loss) AS profit_loss," +
				" COUNT(*) AS trade_count," +
				" COALESCE(SUM(CASE WHEN d.profit_loss > 0 THEN d.profit_loss ELSE 0 END), 0) AS gross_profit," +
				" COALESCE(SUM(CASE WHEN d.profit_loss < 0 THEN d.profit_loss ELSE 0 END), 0) AS gross_loss," +
				" COALESCE(SUM(CASE WHEN d.profit_loss > 0 THEN 1 ELSE 0 END), 0) AS win_count," +
				" COALESCE(SUM(CASE WHEN d.profit_loss < 0 THEN 1 ELSE 0 END), 0) AS loss_count",
		).
		Group("d.ticker_symbol").
		Order("profit_loss DESC")

	if from != nil {
		query = query.Where("d.executed_on >= ?", from.Format("2006-01-02"))
	}
	if to != nil {
		query = query.Where("d.executed_on <= ?", to.Format("2006-01-02"))
	}

	var rows []symbolSummaryRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "DaytradeExecutionRepositoryImpl.AggregateByTickerSymbol error")
	}

	results := make([]*models.DaytradeSymbolSummary, 0, len(rows))
	for _, row := range rows {
		results = append(results, &models.DaytradeSymbolSummary{
			TickerSymbol: row.TickerSymbol,
			BrandName:    row.BrandName,
			ProfitLoss:   row.ProfitLoss,
			TradeCount:   row.TradeCount,
			GrossProfit:  row.GrossProfit,
			GrossLoss:    row.GrossLoss,
			WinCount:     row.WinCount,
			LossCount:    row.LossCount,
		})
	}
	return results, nil
}

func (r *DaytradeExecutionRepositoryImpl) FindByDate(ctx context.Context, date time.Time) ([]*models.DaytradeExecution, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = r.query
	}

	q := tx.DaytradeExecution
	rows, err := q.WithContext(ctx).
		Where(q.ExecutedOn.Eq(date)).
		Order(q.ID.Asc()).
		Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "DaytradeExecutionRepositoryImpl.FindByDate error")
	}

	results := make([]*models.DaytradeExecution, 0, len(rows))
	for _, row := range rows {
		results = append(results, r.convertToDomainModel(row))
	}
	return results, nil
}

type coveredRangeRow struct {
	MinDate sql.NullTime `gorm:"column:min_date"`
	MaxDate sql.NullTime `gorm:"column:max_date"`
}

func (r *DaytradeExecutionRepositoryImpl) GetCoveredRange(ctx context.Context) (*time.Time, *time.Time, error) {
	db, ok := GetTxDB(ctx)
	if !ok {
		db = r.db
	}

	var row coveredRangeRow
	if err := db.WithContext(ctx).
		Table("daytrade_executions").
		Select("MIN(executed_on) AS min_date, MAX(executed_on) AS max_date").
		Scan(&row).Error; err != nil {
		return nil, nil, errors.Wrap(err, "DaytradeExecutionRepositoryImpl.GetCoveredRange error")
	}

	var minDate, maxDate *time.Time
	if row.MinDate.Valid {
		t := row.MinDate.Time
		minDate = &t
	}
	if row.MaxDate.Valid {
		t := row.MaxDate.Time
		maxDate = &t
	}
	return minDate, maxDate, nil
}

func (r *DaytradeExecutionRepositoryImpl) convertToDomainModel(m *genModel.DaytradeExecution) *models.DaytradeExecution {
	return &models.DaytradeExecution{
		ID:           m.ID,
		ExecutedOn:   m.ExecutedOn,
		TradeKind:    m.TradeKind,
		MarginKind:   m.MarginKind,
		TickerSymbol: m.TickerSymbol,
		BrandName:    m.BrandName,
		Quantity:     m.Quantity,
		TradeAmount:  m.TradeAmount,
		UnitPrice:    decimal.NewFromFloat(m.UnitPrice),
		AverageCost:  decimal.NewFromFloat(m.AverageCost),
		ProfitLoss:   m.ProfitLoss,
		OccurrenceNo: m.OccurrenceNo,
		Source:       m.Source,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func (r *DaytradeExecutionRepositoryImpl) convertToDBModel(m *models.DaytradeExecution) *genModel.DaytradeExecution {
	unitPrice, _ := m.UnitPrice.Float64()
	averageCost, _ := m.AverageCost.Float64()
	return &genModel.DaytradeExecution{
		ExecutedOn:   m.ExecutedOn,
		TradeKind:    m.TradeKind,
		MarginKind:   m.MarginKind,
		TickerSymbol: m.TickerSymbol,
		BrandName:    m.BrandName,
		Quantity:     m.Quantity,
		TradeAmount:  m.TradeAmount,
		UnitPrice:    unitPrice,
		AverageCost:  averageCost,
		ProfitLoss:   m.ProfitLoss,
		OccurrenceNo: m.OccurrenceNo,
		Source:       m.Source,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func (r *DaytradeExecutionRepositoryImpl) convertToDBModels(executions []*models.DaytradeExecution) []*genModel.DaytradeExecution {
	rows := make([]*genModel.DaytradeExecution, 0, len(executions))
	for _, e := range executions {
		rows = append(rows, r.convertToDBModel(e))
	}
	return rows
}

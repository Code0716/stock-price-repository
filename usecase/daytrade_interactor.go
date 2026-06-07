//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package usecase

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/usecase/daytrade"
)

type DaytradeInteractor interface {
	ImportSBICsv(ctx context.Context, r io.Reader) (*models.DaytradeImportResult, error)
	GetSummary(ctx context.Context, from, to *time.Time, g models.DaytradeSummaryGranularity) ([]*models.DaytradeSummaryBucket, error)
	GetSummaryByTickerSymbol(ctx context.Context, from, to *time.Time) ([]*models.DaytradeSymbolSummary, error)
	GetExecutionsByDate(ctx context.Context, date time.Time) ([]*models.DaytradeExecution, error)
	GetCoveredRange(ctx context.Context) (minDate, maxDate *time.Time, err error)
	// GetPeriodStats は最大ドローダウン・最大連敗を含む期間統計を返す。from / to は nil 可。
	GetPeriodStats(ctx context.Context, from, to *time.Time) (*models.DaytradePeriodStats, error)
	// GetInsights は大損寄与率・惚れ込み検出を含む反省指標を返す。from / to は nil 可。
	GetInsights(ctx context.Context, from, to *time.Time) (*models.DaytradeInsights, error)
}

type daytradeInteractorImpl struct {
	tx   repositories.Transaction
	repo repositories.DaytradeExecutionRepository
	now  func() time.Time
}

func NewDaytradeInteractor(tx repositories.Transaction, repo repositories.DaytradeExecutionRepository) DaytradeInteractor {
	return &daytradeInteractorImpl{tx: tx, repo: repo, now: time.Now}
}

func (u *daytradeInteractorImpl) ImportSBICsv(ctx context.Context, r io.Reader) (*models.DaytradeImportResult, error) {
	rows, err := daytrade.ParseSBIDaytradeCSV(r, u.now())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &models.DaytradeImportResult{}, nil
	}

	total := len(rows)
	var inserted int
	var deleted int64

	dateSet := make(map[time.Time]struct{})
	for _, row := range rows {
		dateSet[row.ExecutedOn] = struct{}{}
	}
	dates := make([]time.Time, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}

	if err := u.tx.DoInTx(ctx, func(ctx context.Context) error {
		n, err := u.repo.DeleteBySourceAndDates(ctx, "sbi", dates)
		if err != nil {
			return errors.Wrap(err, "DeleteBySourceAndDates error")
		}
		deleted = n
		n2, err := u.repo.BulkInsertIgnore(ctx, rows)
		if err != nil {
			return errors.Wrap(err, "BulkInsertIgnore error")
		}
		inserted = n2
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "DoInTx error")
	}

	return &models.DaytradeImportResult{
		Inserted: inserted,
		Skipped:  total - inserted,
		Deleted:  int(deleted),
		TotalRow: total,
	}, nil
}

func (u *daytradeInteractorImpl) GetSummary(ctx context.Context, from, to *time.Time, g models.DaytradeSummaryGranularity) ([]*models.DaytradeSummaryBucket, error) {
	return u.repo.Aggregate(ctx, from, to, g)
}

func (u *daytradeInteractorImpl) GetSummaryByTickerSymbol(ctx context.Context, from, to *time.Time) ([]*models.DaytradeSymbolSummary, error) {
	return u.repo.AggregateByTickerSymbol(ctx, from, to)
}

func (u *daytradeInteractorImpl) GetExecutionsByDate(ctx context.Context, date time.Time) ([]*models.DaytradeExecution, error) {
	return u.repo.FindByDate(ctx, date)
}

func (u *daytradeInteractorImpl) GetCoveredRange(ctx context.Context) (*time.Time, *time.Time, error) {
	return u.repo.GetCoveredRange(ctx)
}

func (u *daytradeInteractorImpl) GetInsights(ctx context.Context, from, to *time.Time) (*models.DaytradeInsights, error) {
	executions, err := u.repo.FindByDateRange(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "FindByDateRange error")
	}
	trades := daytrade.BuildTradeApprox(executions)
	return daytrade.ComputeInsights(trades), nil
}

func (u *daytradeInteractorImpl) GetPeriodStats(ctx context.Context, from, to *time.Time) (*models.DaytradePeriodStats, error) {
	agg, err := u.repo.AggregateStats(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "AggregateStats error")
	}
	daily, err := u.repo.Aggregate(ctx, from, to, models.DaytradeSummaryGranularityDaily)
	if err != nil {
		return nil, errors.Wrap(err, "Aggregate daily error")
	}
	maxDD, maxRunup, maxStreak := daytrade.ComputeEquityStats(daily)
	return &models.DaytradePeriodStats{
		ProfitLoss:    agg.ProfitLoss,
		TradeCount:    agg.TradeCount,
		GrossProfit:   agg.GrossProfit,
		GrossLoss:     agg.GrossLoss,
		WinCount:      agg.WinCount,
		LossCount:     agg.LossCount,
		MaxProfit:     agg.MaxProfit,
		MaxLoss:       agg.MaxLoss,
		MaxDrawdown:   maxDD,
		MaxRunup:      maxRunup,
		MaxLossStreak: maxStreak,
	}, nil
}

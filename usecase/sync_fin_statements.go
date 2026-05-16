package usecase

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

// SyncFinStatements j-Quantsから指定銘柄の財務情報を取得してDBに保存する
func (si *stockBrandInteractorImpl) SyncFinStatements(ctx context.Context, tickerSymbol string) error {
	infos, err := si.stockAPIClient.GetFinancialStatementsBySymbol(ctx, gateway.StockAPISymbol(tickerSymbol))
	if err != nil {
		return errors.Wrap(err, "GetFinancialStatementsBySymbol error")
	}

	now := time.Now()
	statements := make([]*models.FinStatement, 0, len(infos))
	for _, info := range infos {
		disclosedDate, err := util.FormatStringToDate(info.DisclosedDate)
		if err != nil {
			log.Printf("SyncFinStatements: skip invalid DisclosedDate %q: %v", info.DisclosedDate, err)
			continue
		}

		stmt := &models.FinStatement{
			ID:                  uuid.NewString(),
			TickerSymbol:        info.TickerSymbol,
			DisclosedDate:       disclosedDate,
			TypeOfDocument:      info.TypeOfDocument,
			TypeOfCurrentPeriod: info.TypeOfCurrentPeriod,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		if info.CurrentPeriodEndDate != "" {
			if d, err := util.FormatStringToDate(info.CurrentPeriodEndDate); err == nil {
				stmt.FiscalYearEnd = &d
			}
		}

		stmt.NetSales = parseDecimalPtr(info.NetSales)
		stmt.OperatingProfit = parseDecimalPtr(info.OperatingProfit)
		stmt.OrdinaryProfit = parseDecimalPtr(info.OrdinaryProfit)
		stmt.Profit = parseDecimalPtr(info.Profit)
		stmt.EarningsPerShare = parseDecimalPtr(info.EarningsPerShare)
		stmt.BookValuePerShare = parseDecimalPtr(info.BookValuePerShare)
		stmt.ForecastNetSales = parseDecimalPtr(info.ForecastNetSales)
		stmt.ForecastOperatingProfit = parseDecimalPtr(info.ForecastOperatingProfit)
		stmt.ForecastProfit = parseDecimalPtr(info.ForecastProfit)
		stmt.ForecastEPS = parseDecimalPtr(info.ForecastEarningsPerShare)

		statements = append(statements, stmt)
	}

	if err := si.finStatementRepository.Upsert(ctx, statements); err != nil {
		return errors.Wrap(err, "finStatementRepository.Upsert error")
	}
	return nil
}

func parseDecimalPtr(s string) *decimal.Decimal {
	if s == "" || s == "-" {
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil
	}
	return &d
}

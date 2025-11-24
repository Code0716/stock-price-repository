//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package gateway

import (
	"context"
	"time"
)

type StockAPIClient interface {
	// Yahoo Finance API
	// Yahoo Finance APIから指数/銘柄の情報を取得する（週足以外）。
	GetStockPriceChart(ctx context.Context, symbol StockAPISymbol, interval StockAPIInterval, dateRange StockAPIValidRange) (*StockChartWithRangeAPIResponseInfo, error)
	GetIndexPriceChart(ctx context.Context, symbol StockAPISymbol, interval StockAPIInterval, dateRange StockAPIValidRange) (*StockChartWithRangeAPIResponseInfo, error)
	// Yahoo Finance APIから指数の情報を取得する（週足のみ）。
	GetWeeklyIndexPriceChart(ctx context.Context, symbol StockAPISymbol, dateRange StockAPIValidRange) (*StockChartWithRangeAPIResponseInfo, error)
	// Symbolから決算短信の情報を取得する
	GetBalanceSheetsBySymbol(ctx context.Context, symbol string) (*BalanceSheetsInfo, error)
	GetCurrentStockPriceBySymbol(ctx context.Context, symbol StockAPISymbol, date time.Time) ([]*StockPrice, error)

	// j-Quants
	GetOrSetJQuantsAPIIDTokenToRedis(ctx context.Context) (string, error)
	GetStockBrands(ctx context.Context) ([]*StockBrand, error)
	GetAnnounceFinSchedule(ctx context.Context) ([]*AnnounceFinScheduleResponseInfo, error)
	GetDailyPricesBySymbolAndRange(ctx context.Context, symbol StockAPISymbol, dateFrom, dateTo time.Time) ([]*StockPrice, error)
	GetFinancialStatementsBySymbol(ctx context.Context, symbol StockAPISymbol) ([]*FinancialStatementsResponseInfo, error)
	GetFinancialStatementsByDate(ctx context.Context, date time.Time) ([]*FinancialStatementsResponseInfo, error)
	GetTradingCalendarsInfo(ctx context.Context, filter TradingCalendarsInfoFilter) ([]*TradingCalendarsInfo, error)
}

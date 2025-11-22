//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
)

type StockAPIClient struct {
	request     HTTPRequest
	redisClient *redis.Client
	// reteLimiter *rate.Limiter
}

func NewStockAPIClient(
	request HTTPRequest,
	redisClient *redis.Client,
) gateway.StockAPIClient {
	// reteLimiter := rate.NewLimiter(rate.Every(2*time.Second), 1)
	return &StockAPIClient{
		request:     request,
		redisClient: redisClient,
		// reteLimiter: reteLimiter,
	}
}

// waitLimit リクエスト回数を制限
// func (c *StockAPIClient) waitLimit(ctx context.Context) error {
// 	err := c.reteLimiter.Wait(ctx)
// 	if err != nil {
// 		return errors.Wrap(err, "rate.NewLimiter.Wait error")
// 	}
// 	return nil
// }

func (c *StockAPIClient) GetStockPriceChart(ctx context.Context, symbol gateway.StockAPISymbol, interval gateway.StockAPIInterval, dateRange gateway.StockAPIValidRange) (*gateway.StockChartWithRangeAPIResponseInfo, error) {
	tickerSymbol := c.getYahooFinanceAPIStckBrandSymbol(symbol.String())
	return c.getStockPriceChart(ctx, tickerSymbol, interval, dateRange)
}

func (c *StockAPIClient) GetIndexPriceChart(ctx context.Context, symbol gateway.StockAPISymbol, interval gateway.StockAPIInterval, dateRange gateway.StockAPIValidRange) (*gateway.StockChartWithRangeAPIResponseInfo, error) {
	tickerSymbol := c.getYahooFinanceAPIIndexBrandSymbol(symbol.String())
	return c.getStockPriceChart(ctx, tickerSymbol, interval, dateRange)
}

// 現在値の取得
func (c *StockAPIClient) GetCurrentStockPriceBySymbol(ctx context.Context, symbol gateway.StockAPISymbol, date time.Time) ([]*gateway.StockPrice, error) {
	// j-Quants 場中の値段は取得できないようだ。
	// 無理して使わなくていいと思う
	return c.getCurrentStockPriceBySymbolYF(ctx, symbol.String())
}

func (c *StockAPIClient) GetDailyPricesBySymbolAndRange(ctx context.Context, symbol gateway.StockAPISymbol, dateFrom, dateTo time.Time) ([]*gateway.StockPrice, error) {
	return c.getDailyPricesBySymbolAndRangeJQ(ctx, symbol.String(), dateFrom, dateTo)
}

// GetFinancialStatementsBySymbol シンボルから財務情報の取得
func (c *StockAPIClient) GetFinancialStatementsBySymbol(ctx context.Context, symbol gateway.StockAPISymbol) ([]*gateway.FinancialStatementsResponseInfo, error) {
	response, err := c.getFinancialStatementsJQ(ctx, symbol.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetFinancialStatementsBySymbol error")
	}
	return response, nil
}

// GetFinancialStatementsByDate dateから財務情報の取得
func (c *StockAPIClient) GetFinancialStatementsByDate(ctx context.Context, date time.Time) ([]*gateway.FinancialStatementsResponseInfo, error) {
	response, err := c.getFinancialStatementsJQ(ctx, "", util.ToPtrGenerics(date))
	if err != nil {
		return nil, errors.Wrap(err, "GetFinancialStatementsByDate error")
	}
	return response, nil
}

// GetTradingCalendarsInfo 期間内の営業日を取得する。
func (c *StockAPIClient) GetTradingCalendarsInfo(ctx context.Context, filter gateway.TradingCalendarsInfoFilter) ([]*gateway.TradingCalendarsInfo, error) {
	results, err := c.getTradingCalendarsInfo(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "GetTradingCalendarsInfo error")
	}

	return results, nil
}

package models

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	AnalyzeStockBrandPriceHistoryMethodSector25          string = "analyze_stock_brand_price_by_sector: 25日"
	AnalyzeStockBrandPriceHistoryMethodSector75          string = "analyze_stock_brand_price_by_sector: 75日"
	AnalyzeStockBrandPriceHistoryMethodNikkei25          string = "analyze_stock_brand_price_by_nikkei: 25日"
	AnalyzeStockBrandPriceHistoryMethodNikkei75          string = "analyze_stock_brand_price_by_nikkei: 75日"
	AnalyzeStockBrandPriceHistoryMethodFindMACDBullishV1       string = "find_macd_bullish_stock_v1"
	AnalyzeStockBrandPriceHistoryMethodFindTriangleV1          string = "find_triangle_formation_stock_v1"
	AnalyzeStockBrandPriceHistoryMethodFindBollingerBreakoutV1 string = "find_bollinger_breakout_stock_v1"
	AnalyzeStockBrandPriceHistoryMethodFindMLRankedV1          string = "find_ml_ranked_stocks_v1"
	AnalyzeStockBrandPriceHistoryActionBuy                     string = "Buy"
	AnalyzeStockBrandPriceHistoryActionSell                    string = "Sell"

	AnalyzeStockBrandPriceHistorySortByCreatedAt  string = "created_at"
	AnalyzeStockBrandPriceHistorySortByProfit     string = "profit"
	AnalyzeStockBrandPriceHistorySortByProfitRate string = "profit_rate"
	AnalyzeStockBrandPriceHistoryOrderAsc         string = "asc"
	AnalyzeStockBrandPriceHistoryOrderDesc        string = "desc"
)

type AnalyzeStockBrandPriceHistory struct {
	ID              string          `json:"id"`
	StockBrandID    string          `json:"stockBrandId"`
	Name            string          `json:"name"`
	TickerSymbol    string          `json:"tickerSymbol"`
	TradePrice      decimal.Decimal `json:"tradePrice"`
	CurrentPrice    decimal.Decimal `json:"currentPrice"`
	PriceDifference decimal.Decimal `json:"priceDifference"`
	Action          string          `json:"action"`
	Method          string          `json:"method"`
	Memo            *string         `json:"memo"`
	CreatedAt       time.Time       `json:"createdAt"`
}

type AnalyzeStockBrandPriceHistoryFilter struct {
	TickerSymbol string
	Action       string
	Method       string
	SortBy       string
	Order        string
	Page         int
	Limit        int
}

type PaginatedAnalyzeStockBrandPriceHistories struct {
	Histories  []*AnalyzeStockBrandPriceHistory
	Page       int
	Limit      int
	Total      int64
	TotalPages int
}

func NewAnalyzeStockBrandPriceHistory(
	id string, // uuid
	stockBrandID string, // 銘柄IDs
	name string, // 銘柄名
	tickerSymbol string, // 証券コード
	tradePrice decimal.Decimal, // トレード金額
	currentPrice decimal.Decimal, // 現在値
	action string, // 売り/買いの別
	method string, // 分析方法
	memo *string, // メモ
	createdAt time.Time, // created_at
) *AnalyzeStockBrandPriceHistory {
	return &AnalyzeStockBrandPriceHistory{
		ID:              id,
		StockBrandID:    stockBrandID,
		Name:            name,
		TickerSymbol:    tickerSymbol,
		TradePrice:      tradePrice,
		CurrentPrice:    currentPrice,
		PriceDifference: currentPrice.Sub(tradePrice),
		Action:          action,
		Method:          method,
		Memo:            memo,
		CreatedAt:       createdAt,
	}
}

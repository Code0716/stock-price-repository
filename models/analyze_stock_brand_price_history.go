package models

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	AnalyzeStockBrandPriceHistoryMethodSector25 string = "analyze_stock_brand_price_by_sector: 25日"
	AnalyzeStockBrandPriceHistoryMethodSector75 string = "analyze_stock_brand_price_by_sector: 75日"
	AnalyzeStockBrandPriceHistoryMethodNikkei25 string = "analyze_stock_brand_price_by_nikkei: 25日"
	AnalyzeStockBrandPriceHistoryMethodNikkei75 string = "analyze_stock_brand_price_by_nikkei: 75日"
	AnalyzeStockBrandPriceHistoryActionBuy      string = "Buy"
	AnalyzeStockBrandPriceHistoryActionSell     string = "Sell"
)

type AnalyzeStockBrandPriceHistory struct {
	ID           string          // uuid
	StockBrandID string          // 銘柄IDs
	Name         string          // 銘柄名
	TickerSymbol string          // 証券コード
	TradePrice   decimal.Decimal // トレード金額
	CurrentPrice decimal.Decimal // 現在値
	Action       string          // 売り/買いの別
	Method       string          // 分析方法
	Memo         *string         // メモ
	CreatedAt    time.Time       // created_at
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
		ID:           id,
		StockBrandID: stockBrandID,
		Name:         name,
		TickerSymbol: tickerSymbol,
		TradePrice:   tradePrice,
		CurrentPrice: currentPrice,
		Action:       action,
		Method:       method,
		Memo:         memo,
		CreatedAt:    createdAt,
	}
}

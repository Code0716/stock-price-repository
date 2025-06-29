package gateway

import (
	"time"

	"github.com/shopspring/decimal"
)

type StockAPIInterval string

// 1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo
const (
	StockAPIInterval1M  StockAPIInterval = "1m"
	StockAPIInterval2M  StockAPIInterval = "2m"
	StockAPIInterval5M  StockAPIInterval = "5m"
	StockAPIInterval15M StockAPIInterval = "15m"
	StockAPIInterval30M StockAPIInterval = "30m"
	StockAPIInterval60M StockAPIInterval = "60m"
	StockAPIInterval90M StockAPIInterval = "90m"
	StockAPIInterval1H  StockAPIInterval = "1h"
	StockAPIInterval1D  StockAPIInterval = "1d"
	StockAPIInterval5D  StockAPIInterval = "5d"
	StockAPIInterval1WK StockAPIInterval = "1wk"
	StockAPIInterval1MO StockAPIInterval = "1mo"
	StockAPIInterval3MO StockAPIInterval = "3mo"
)

func (i StockAPIInterval) String() string {
	return string(i)
}

type StockAPIValidRange string

const (
	StockAPIValidRange1D  StockAPIValidRange = "1d"
	StockAPIValidRange5D  StockAPIValidRange = "5d"
	StockAPIValidRange1MO StockAPIValidRange = "1mo"
	StockAPIValidRange3MO StockAPIValidRange = "3mo"
	StockAPIValidRange6MO StockAPIValidRange = "6mo"
	StockAPIValidRange1Y  StockAPIValidRange = "1y"
	StockAPIValidRange2Y  StockAPIValidRange = "2y"
	StockAPIValidRange5Y  StockAPIValidRange = "5y"
	StockAPIValidRange10Y StockAPIValidRange = "10y"
	StockAPIValidRangeYTD StockAPIValidRange = "ytd"
	StockAPIValidRangeMAX StockAPIValidRange = "max"
)

func (r StockAPIValidRange) String() string {
	return string(r)
}

type StockAPISymbol string

const (
	StockAPISymbolNikkei StockAPISymbol = "^N225"
	StockAPISymbolDji    StockAPISymbol = "^DJI"
)

func (s StockAPISymbol) String() string {
	return string(s)
}

type StockChartWithRangeAPIResponseInfo struct {
	TickerSymbol    string
	InstrumentType  string        // 指数/銘柄の別
	DataGranularity string        // データの粒度
	Range           string        // 期間の幅
	Indicator       []*StockPrice // インジケータ
}

type BalanceSheetsInfo struct {
	TickerSymbol  string
	BalanceSheets []*BalanceSheetItem
	Calendar      *Calendar
}

type Calendar struct {
	ExDividendDate  *time.Time
	EarningsDate    []*time.Time
	EarningsHigh    decimal.Decimal
	EarningsLow     decimal.Decimal
	EarningsAverage decimal.Decimal
	RevenueHigh     int64
	RevenueLow      int64
	RevenueAverage  int64
}

// // TradingCalendarsInfo 相場の営業日の情報
type TradingCalendarsInfo struct {
	Date            time.Time
	HolidayDivision string
}

// TradingCalendarsInfoFilter カレンダーのフィルター
type TradingCalendarsInfoFilter struct {
	From time.Time
	To   time.Time
}

type TradingCalendarStatus int

const (
	TradingCalendarStatusClose          TradingCalendarStatus = iota // 非営業日
	TradingCalendarStatusOpen                                        // 営業日
	TradingCalendarStatusHalfDay                                     // 半営業日(前場のみ営業)
	TradingCalendarStatusHolidayButOpen                              // 祝日だが先物は営業している日
)

package models

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	NFLeverageETFTickerSymbol      string = "1570"
	NFDoubleInverseETFTickerSymbol string = "1357"
)

type StockBrand struct {
	ID               string    `json:"id"`
	TickerSymbol     string    `json:"tickerSymbol"`
	Name             string    `json:"name"`
	MarketCode       string    `json:"marketCode"`
	MarketName       string    `json:"marketName"`
	Sector33Code     string    `json:"sector33Code"`
	Sector33CodeName string    `json:"sector33CodeName"`
	Sector17Code     string    `json:"sector17Code"`
	Sector17CodeName string    `json:"sector17CodeName"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// PaginatedStockBrands ページネーション付き銘柄一覧
type PaginatedStockBrands struct {
	Brands     []*StockBrand
	NextCursor *string
	Limit      int
}

// StockBrandFilter 銘柄検索のフィルタ条件
type StockBrandFilter struct {
	// OnlyMainMarkets trueの場合、主要市場（111, 112, 113）のみ取得
	OnlyMainMarkets bool
	// MarketCodes 指定された市場コードでフィルタ（nilの場合は全市場）
	// OnlyMainMarketsがtrueの場合、このフィールドは無視される
	MarketCodes []string
	// SymbolFrom ページネーション用の開始シンボル（空文字列の場合は最初から）
	SymbolFrom string
	// Limit 取得件数上限（0の場合は全件取得）
	Limit int
}

// NewStockBrandFilter デフォルトのフィルタを作成
func NewStockBrandFilter() *StockBrandFilter {
	return &StockBrandFilter{
		OnlyMainMarkets: false,
		MarketCodes:     nil,
		SymbolFrom:      "",
		Limit:           0,
	}
}

// WithOnlyMainMarkets 主要市場のみに絞り込む
func (f *StockBrandFilter) WithOnlyMainMarkets() *StockBrandFilter {
	f.OnlyMainMarkets = true
	return f
}

// WithMarketCodes 指定市場コードに絞り込む
func (f *StockBrandFilter) WithMarketCodes(codes ...string) *StockBrandFilter {
	f.MarketCodes = codes
	return f
}

// WithPagination ページネーション設定
func (f *StockBrandFilter) WithPagination(symbolFrom string, limit int) *StockBrandFilter {
	f.SymbolFrom = symbolFrom
	f.Limit = limit
	return f
}

type StockBrandWithVolumeAverage struct {
	Name          string
	TickerSymbol  string
	VolumeAverage int64
	CreatedAt     time.Time
}

func NewStockBrandWithVolumeAverage(
	tickerSymbol string,
	volumeAverage int64,
	createdAt time.Time,
) *StockBrandWithVolumeAverage {
	return &StockBrandWithVolumeAverage{
		TickerSymbol:  tickerSymbol,
		VolumeAverage: volumeAverage,
		CreatedAt:     createdAt,
	}
}

type StockBrandSignals struct {
	Date         time.Time
	TickerSymbol string
	Name         string // 銘柄名
	// GoldenCrossLitUp bool // ゴールデンクロス点灯
	MovingAverage StockBrandSignalsMovingAverage
	Volume        StockBrandSignalsVolume
	Volatility    StockBrandVolatility
}

type StockBrandSignalsMovingAverage struct {
	CurrentClosePrice                decimal.Decimal
	Above5Day                        bool
	Above25Day                       bool
	Above75Day                       bool
	SupportAndResistanceLine         StockBrandSignalSupportAndResistanceLine
	Above5DayByLast3DaysClosingPrice bool // 直近3間の終値が5日移動平均線を超えているか
}

type StockBrandSignalsVolume struct {
	Rising                               bool
	Skyrocketed                          bool // 出来高急上昇
	AboveMedianForLast2MonthsByLast3Days bool //　直近2ヶ月の出来高中央値を3日間で超えてる。
}

type StockBrandSignalSupportAndResistanceLine struct {
	SupportLine    StockBrandSignalSupportAndResistanceLineItem
	ResistanceLine StockBrandSignalSupportAndResistanceLineItem
}

type StockBrandSignalSupportAndResistanceLineItem struct {
	FirstPeriod       decimal.Decimal
	AboveFirstPeriod  bool
	SecondPeriod      decimal.Decimal
	AboveSecondPeriod bool
	ThirdPeriod       decimal.Decimal
	AboveThirdPeriod  bool
}

type StockBrandVolatility struct {
	HighVolatility             bool
	ExceededStandardValueTimes int
}

func NewStockBrandSignals(
	date time.Time,
	tickerSymbol string,
	name string,
	movingAverage StockBrandSignalsMovingAverage,
	volume StockBrandSignalsVolume,
	volatility StockBrandVolatility,
) *StockBrandSignals {
	return &StockBrandSignals{
		Date:          date,
		TickerSymbol:  tickerSymbol,
		Name:          name,
		MovingAverage: movingAverage,
		Volume:        volume,
		Volatility:    volatility,
	}
}

func NewStockBrand(
	tickerSymbol string,
	Name string,
	marketCode string,
	marketName string,
	sector33Code string,
	sector33CodeName string,
	sector17Code string,
	sector17CodeName string,
	CreatedAt time.Time,
	UpdatedAt time.Time,
) *StockBrand {
	return &StockBrand{
		TickerSymbol:     tickerSymbol,
		Name:             Name,
		MarketCode:       marketCode,
		MarketName:       marketName,
		Sector33Code:     sector33Code,
		Sector33CodeName: sector33CodeName,
		Sector17Code:     sector17Code,
		Sector17CodeName: sector17CodeName,
		CreatedAt:        CreatedAt,
		UpdatedAt:        UpdatedAt,
	}
}

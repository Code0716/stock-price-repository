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
	ID               string    // ID
	TickerSymbol     string    // 証券コード
	Name             string    // 銘柄名
	MarketCode       string    // 市場コード
	MarketName       string    // 市場名
	Sector33Code     string    // 33業種コード
	Sector33CodeName string    // 33業種区分
	Sector17Code     string    // 17業種コード
	Sector17CodeName string    // 17業種区分
	CreatedAt        time.Time // created_at
	UpdatedAt        time.Time // deleted_at
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

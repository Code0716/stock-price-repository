package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// StockBrandDailyPriceForAnalyze
type StockBrandDailyPriceForAnalyze struct {
	ID           string          `json:"id"`
	TickerSymbol string          `json:"tickerSymbol"`
	Date         time.Time       `json:"date"`
	High         decimal.Decimal `json:"high"`
	Low          decimal.Decimal `json:"low"`
	Open         decimal.Decimal `json:"open"`
	Close        decimal.Decimal `json:"close"`
	Volume       int64           `json:"volume"`
	Adjclose     decimal.Decimal `json:"adjClose"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

func NewStockBrandDailyPriceForAnalyze(
	id string,
	date time.Time,
	tickerSymbol string,
	high decimal.Decimal,
	low decimal.Decimal,
	open decimal.Decimal,
	closePrice decimal.Decimal,
	volume int64,
	adjclose decimal.Decimal,
	createdAt time.Time,
	updatedAt time.Time,
) *StockBrandDailyPriceForAnalyze {
	return &StockBrandDailyPriceForAnalyze{
		ID:           id,
		Date:         date,
		TickerSymbol: tickerSymbol,
		High:         high,
		Low:          low,
		Open:         open,
		Close:        closePrice,
		Volume:       volume,
		Adjclose:     adjclose,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

// AdjustForSplit adjusts the stock price for a stock split.
func (s *StockBrandDailyPriceForAnalyze) AdjustForSplit(splitRatio decimal.Decimal) *StockBrandDailyPriceForAnalyze {
	newVolumeDecimal := decimal.NewFromInt(s.Volume).Mul(splitRatio)

	return NewStockBrandDailyPriceForAnalyze(
		s.ID,
		s.Date,
		s.TickerSymbol,
		s.High.Div(splitRatio),
		s.Low.Div(splitRatio),
		s.Open.Div(splitRatio),
		s.Close.Div(splitRatio),
		newVolumeDecimal.IntPart(),
		s.Adjclose.Div(splitRatio),
		s.CreatedAt,
		time.Now(),
	)
}

// AdjustForConsolidation adjusts the stock price for a stock consolidation (reverse split).
// consolidationRatio は「旧株数 / 新株数」（例: 5株を1株に併合する場合は 5）。
// 価格は ratio 倍、出来高は 1/ratio になる。
func (s *StockBrandDailyPriceForAnalyze) AdjustForConsolidation(consolidationRatio decimal.Decimal) *StockBrandDailyPriceForAnalyze {
	newVolumeDecimal := decimal.NewFromInt(s.Volume).Div(consolidationRatio)

	return NewStockBrandDailyPriceForAnalyze(
		s.ID,
		s.Date,
		s.TickerSymbol,
		s.High.Mul(consolidationRatio),
		s.Low.Mul(consolidationRatio),
		s.Open.Mul(consolidationRatio),
		s.Close.Mul(consolidationRatio),
		newVolumeDecimal.IntPart(),
		s.Adjclose.Mul(consolidationRatio),
		s.CreatedAt,
		time.Now(),
	)
}

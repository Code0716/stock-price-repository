package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/shopspring/decimal"
)

// AdjustHistoricalDataForStockSplit Input input
type AdjustHistoricalDataForStockSplit interface {
	AdjustHistoricalDataForStockSplit(ctx context.Context, code string, splitDate time.Time, splitRatio decimal.Decimal, dryRun bool) error
}

// AdjustHistoricalDataForStockSplitInteractor Interactor
type AdjustHistoricalDataForStockSplitInteractor struct {
	StockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
}

// NewAdjustHistoricalDataForStockSplit New
func NewAdjustHistoricalDataForStockSplit(
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
) AdjustHistoricalDataForStockSplit {
	return &AdjustHistoricalDataForStockSplitInteractor{
		StockBrandsDailyPriceForAnalyzeRepository: stockBrandsDailyPriceForAnalyzeRepository,
	}
}

// AdjustHistoricalDataForStockSplit adjust historical data for stock split
func (ui *AdjustHistoricalDataForStockSplitInteractor) AdjustHistoricalDataForStockSplit(
	ctx context.Context,
	code string,
	splitDate time.Time,
	splitRatio decimal.Decimal,
	dryRun bool,
) error {
	// 指定された日付以前のデータを取得する
	// 分割日そのものも、分割後の価格で反映されるべきなのか、分割前の価格なのかはケースバイケースだが、
	// 通常「分割した日付」のデータは既に分割後の価格で市場取引されているはず。
	// なので、修正すべきは「分割日の前日」まで。
	// 指示では「inputするデータは、...分割した日付...」とあり、「分割日以前の過去データを」修正する。
	// 通常、効力発生日の前日までのデータが修正対象。
	// ここでは splitDate より前の日付 ( < splitDate ) を対象とするため、 DateTo に splitDate の前日を設定する。
	// ただし、ListDailyPricesBySymbolFilter の DateTo は inclusive (Lte) なので、
	// splitDate.AddDate(0, 0, -1) をセットする。

	targetDateTo := splitDate.AddDate(0, 0, -1)

	analyzeDailyPrices, err := ui.StockBrandsDailyPriceForAnalyzeRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: code,
		DateTo:       &targetDateTo,
	})
	if err != nil {
		return err
	}

	if len(analyzeDailyPrices) == 0 {
		fmt.Printf("No historical data found for code: %s before %v in StockBrandsDailyPriceForAnalyze\n", code, splitDate)
	} else {
		var updateAnalyzePrices []*models.StockBrandDailyPriceForAnalyze

		for _, price := range analyzeDailyPrices {
			newOpen := price.Open.Div(splitRatio)
			newClose := price.Close.Div(splitRatio)
			newHigh := price.High.Div(splitRatio)
			newLow := price.Low.Div(splitRatio)
			newAdjClose := price.Adjclose.Div(splitRatio)

			newVolumeDecimal := decimal.NewFromInt(price.Volume).Mul(splitRatio)
			newVolume := newVolumeDecimal.IntPart()

			if dryRun {
				fmt.Printf("[StockBrandDailyPriceForAnalyze] DryRun: %s Date: %s Open: %s -> %s, Close: %s -> %s, Volume: %d -> %d\n",
					code, price.Date.Format("2006-01-02"),
					price.Open.String(), newOpen.String(),
					price.Close.String(), newClose.String(),
					price.Volume, newVolume,
				)
			} else {
				price.Open = newOpen
				price.Close = newClose
				price.High = newHigh
				price.Low = newLow
				price.Adjclose = newAdjClose
				price.Volume = newVolume
				price.UpdatedAt = time.Now()

				updateAnalyzePrices = append(updateAnalyzePrices, price)
			}
		}

		if !dryRun && len(updateAnalyzePrices) > 0 {
			err = ui.StockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze(ctx, updateAnalyzePrices)
			if err != nil {
				return err
			}
			fmt.Printf("Successfully updated %d records for code: %s in StockBrandsDailyPriceForAnalyze\n", len(updateAnalyzePrices), code)
		}
	}

	if dryRun {
		fmt.Printf("DryRun finished.\n")
	}

	return nil
}

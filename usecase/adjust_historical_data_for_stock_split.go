package usecase

import (
	"context"
	"fmt"
	"log"
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
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
	appliedStockSplitsHistoryRepository       repositories.AppliedStockSplitsHistoryRepository
}

// NewAdjustHistoricalDataForStockSplit New
func NewAdjustHistoricalDataForStockSplit(
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	appliedStockSplitsHistoryRepository repositories.AppliedStockSplitsHistoryRepository,
) AdjustHistoricalDataForStockSplit {
	return &AdjustHistoricalDataForStockSplitInteractor{
		stockBrandsDailyPriceForAnalyzeRepository: stockBrandsDailyPriceForAnalyzeRepository,
		appliedStockSplitsHistoryRepository:       appliedStockSplitsHistoryRepository,
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

	// 株式分割履歴から実行済みか確認する。
	exists, err := ui.appliedStockSplitsHistoryRepository.Exists(ctx, code, splitDate)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Stock split adjustment already applied for code: %s on date: %v", code, splitDate)
		return nil
	}

	analyzeDailyPrices, err := ui.stockBrandsDailyPriceForAnalyzeRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: code,
		DateTo:       &targetDateTo,
	})
	if err != nil {
		return err
	}

	if len(analyzeDailyPrices) == 0 {
		log.Printf("No historical data found for code: %s before %v in StockBrandsDailyPriceForAnalyze", code, splitDate)
		return nil
	}

	err = ui.updateAnalyzePrices(ctx, analyzeDailyPrices, code, splitDate, splitRatio, dryRun)
	if err != nil {
		return err
	}

	if !dryRun {

		// 株式分割履歴を登録する
		history := models.NewAppliedStockSplitHistory(
			code,
			splitDate,
			splitRatio,
		)
		err = ui.appliedStockSplitsHistoryRepository.Create(ctx, history)
		if err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Printf("DryRun finished.\n")
	}

	return nil
}

// updateAnalyzePrices 分割後の価格に調整する
func (ui *AdjustHistoricalDataForStockSplitInteractor) updateAnalyzePrices(
	ctx context.Context,
	analyzeDailyPrices []*models.StockBrandDailyPriceForAnalyze,
	code string,
	splitDate time.Time,
	splitRatio decimal.Decimal,
	dryRun bool,
) error {

	if len(analyzeDailyPrices) == 0 {
		log.Printf("No historical data found for code: %s before %v in StockBrandsDailyPriceForAnalyze", code, splitDate)
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
			// TODO: models.NewStockBrandDailyPriceForAnalyzeを使うように修正する。
			newPrice := models.NewStockBrandDailyPriceForAnalyze(
				price.ID,
				price.Date,
				price.TickerSymbol,
				newHigh,
				newLow,
				newOpen,
				newClose,
				newVolume,
				newAdjClose,
				price.CreatedAt,
				time.Now(),
			)

			if dryRun {
				log.Printf("[StockBrandDailyPriceForAnalyze] DryRun: %s Date: %s Open: %s -> %s, Close: %s -> %s, Volume: %d -> %d",
					code, price.Date.Format("2006-01-02"),
					price.Open.String(), newPrice.Open.String(),
					price.Close.String(), newPrice.Close.String(),
					price.Volume, newPrice.Volume,
				)
			} else {
				updateAnalyzePrices = append(updateAnalyzePrices, newPrice)
			}
		}

		if !dryRun && len(updateAnalyzePrices) > 0 {
			err := ui.stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze(ctx, updateAnalyzePrices)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

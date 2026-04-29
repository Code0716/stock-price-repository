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

// AdjustHistoricalDataForStockConsolidation Input input
type AdjustHistoricalDataForStockConsolidation interface {
	AdjustHistoricalDataForStockConsolidation(ctx context.Context, code string, consolidationDate time.Time, consolidationRatio decimal.Decimal, dryRun bool) error
}

// AdjustHistoricalDataForStockConsolidationInteractor Interactor
type AdjustHistoricalDataForStockConsolidationInteractor struct {
	stockBrandsDailyPriceForAnalyzeRepository   repositories.StockBrandsDailyPriceForAnalyzeRepository
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository
}

// NewAdjustHistoricalDataForStockConsolidation New
func NewAdjustHistoricalDataForStockConsolidation(
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	appliedStockConsolidationsHistoryRepository repositories.AppliedStockConsolidationsHistoryRepository,
) AdjustHistoricalDataForStockConsolidation {
	return &AdjustHistoricalDataForStockConsolidationInteractor{
		stockBrandsDailyPriceForAnalyzeRepository:   stockBrandsDailyPriceForAnalyzeRepository,
		appliedStockConsolidationsHistoryRepository: appliedStockConsolidationsHistoryRepository,
	}
}

// AdjustHistoricalDataForStockConsolidation adjust historical data for stock consolidation (reverse split).
// consolidationRatio は「旧株数 / 新株数」（例: 5株を1株に併合する場合は 5）。
// 価格を ratio 倍、出来高を 1/ratio に補正する。
func (ui *AdjustHistoricalDataForStockConsolidationInteractor) AdjustHistoricalDataForStockConsolidation(
	ctx context.Context,
	code string,
	consolidationDate time.Time,
	consolidationRatio decimal.Decimal,
	dryRun bool,
) error {
	// 指定された日付が未来の場合は処理をスキップする
	if consolidationDate.After(time.Now()) {
		log.Printf("Consolidation date %v is in the future. Skipping adjustment.", consolidationDate)
		return nil
	}

	// 効力発生日のデータは既に併合後の価格で取引されているはずなので、
	// 修正対象は併合日の前日まで。ListDailyPricesBySymbolFilter の DateTo は inclusive (Lte) なので、
	// consolidationDate.AddDate(0, 0, -1) をセットする。
	targetDateTo := consolidationDate.AddDate(0, 0, -1)

	// 株式併合履歴から実行済みか確認する。
	exists, err := ui.appliedStockConsolidationsHistoryRepository.Exists(ctx, code, consolidationDate)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Stock consolidation adjustment already applied for code: %s on date: %v", code, consolidationDate)
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
		log.Printf("No historical data found for code: %s before %v in StockBrandsDailyPriceForAnalyze", code, consolidationDate)
		return nil
	}

	err = ui.updateAnalyzePrices(ctx, analyzeDailyPrices, code, consolidationDate, consolidationRatio, dryRun)
	if err != nil {
		return err
	}

	if !dryRun {
		// 株式併合履歴を登録する
		history := models.NewAppliedStockConsolidationHistory(
			code,
			consolidationDate,
			consolidationRatio,
		)
		err = ui.appliedStockConsolidationsHistoryRepository.Create(ctx, history)
		if err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Printf("DryRun finished.\n")
	}

	return nil
}

// updateAnalyzePrices 併合後の価格に調整する
func (ui *AdjustHistoricalDataForStockConsolidationInteractor) updateAnalyzePrices(
	ctx context.Context,
	analyzeDailyPrices []*models.StockBrandDailyPriceForAnalyze,
	code string,
	consolidationDate time.Time,
	consolidationRatio decimal.Decimal,
	dryRun bool,
) error {

	if len(analyzeDailyPrices) == 0 {
		log.Printf("No historical data found for code: %s before %v in StockBrandsDailyPriceForAnalyze", code, consolidationDate)
	} else {
		var updateAnalyzePrices []*models.StockBrandDailyPriceForAnalyze
		for _, price := range analyzeDailyPrices {
			newPrice := price.AdjustForConsolidation(consolidationRatio)

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

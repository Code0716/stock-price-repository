package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

// CreateDailyStockPrice - 全銘柄の日足を取得して保存する
func (si *stockBrandsDailyStockPriceInteractorImpl) CreateDailyStockPrice(ctx context.Context, now time.Time) error {
	//  直近5日分の日足を作成
	for i := range 5 {
		if err := si.createDailyStockPrice(ctx, now.AddDate(0, 0, -i)); err != nil {
			return errors.Wrap(err, "createDailyStockPrice error")
		}
	}
	return nil
}

// createDailyStockPrice - 日足を作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) createDailyStockPrice(ctx context.Context, now time.Time) error {
	err := si.tx.DoInTx(ctx, func(ctx context.Context) error {
		// 銘柄を取得
		currentBrands, err := si.stockBrandRepository.FindAll(ctx)
		if err != nil {
			return errors.Wrap(err, "stockBrandRepository.FindAll error")
		}

		currentBrandsMap := make(map[string]*models.StockBrand, len(currentBrands))
		for _, v := range currentBrands {
			currentBrandsMap[v.TickerSymbol] = v
		}

		// 全銘柄の日足を作成
		stockPricesWithBrand := si.newStockBrandDailyPrices(ctx, currentBrandsMap, now)
		if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, stockPricesWithBrand); err != nil {
			return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.CreateMany error")
		}

		if err := si.stockBrandsDailyPriceForAnalyzeRepository.
			CreateStockBrandDailyPriceForAnalyze(
				ctx,
				si.newStockBrandDailyPriceForAnalyzeByStockBrandsDailyPrice(stockPricesWithBrand, now),
			); err != nil {
			return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze error")
		}

		// 3年前の日付を計算
		threeYearsAgo := now.AddDate(-3, 0, 0)
		if err := si.stockBrandsDailyPriceForAnalyzeRepository.DeleteBeforeDate(ctx, threeYearsAgo); err != nil {
			return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.DeleteBeforeDate error")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	return nil
}

// createDailyStockPrices - 全銘柄の一日の日足スライスを作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPrices(ctx context.Context, currentBrandsMap map[string]*models.StockBrand, now time.Time) []*models.StockBrandDailyPrice {
	stockPrices, err := si.stockAPIClient.GetAllBrandDailyPricesByDate(ctx, now)
	if err != nil {
		return nil
	}

	if stockPrices == nil {
		return nil
	}

	var result []*models.StockBrandDailyPrice
	for _, v := range stockPrices {
		price := si.newStockBrandDailyPrice(currentBrandsMap[v.TickerSymbol], v, now)
		if price == nil {
			continue
		}
		result = append(result, price)
	}

	return result
}

// newStockBrandDailyPrice - StockBrandDailyPrice 作成
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPrice(stockBrand *models.StockBrand, prices *gateway.StockPrice, now time.Time) *models.StockBrandDailyPrice {
	if stockBrand == nil {
		return nil
	}

	if prices.High.IsZero() && prices.Close.IsZero() && prices.Low.IsZero() && prices.Open.IsZero() {
		return nil
	}

	result := models.NewStockBrandDailyPrice(
		util.GenerateUUID(),
		stockBrand.ID,
		prices.Date,
		prices.TickerSymbol,
		prices.High,
		prices.Low,
		prices.Open,
		prices.Close,
		prices.Volume,
		prices.AdjustmentClose,
		now,
		now,
	)

	return result
}

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
	// applyCorporateActionsForAnalyze で使うため、DoInTxの外からも参照できるようにしておく
	var gatewayPrices []*gateway.StockPrice

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

		// J-Quantsから当日分の日足を1回だけ取得し、raw用・分析用の両方の変換に使う
		var apiErr error
		gatewayPrices, apiErr = si.stockAPIClient.GetAllBrandDailyPricesByDate(ctx, now)
		if apiErr != nil {
			return errors.Wrap(apiErr, "stockAPIClient.GetAllBrandDailyPricesByDate error")
		}

		// 全銘柄の日足を作成（素値。これまで通り保持する）
		stockPricesWithBrand := si.newStockBrandDailyPricesFromGateway(currentBrandsMap, gatewayPrices, now)
		if err := si.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, stockPricesWithBrand); err != nil {
			return errors.Wrap(err, "stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice error")
		}

		// 分析用は素値ではなくJ-Quantsの分割・併合調整済みOHLCV（Adjustment*）を保存する。
		// クイズ・分析はこちらを読む前提のテーブルのため、素値に基づく再計算はしない。
		if err := si.stockBrandsDailyPriceForAnalyzeRepository.
			CreateStockBrandDailyPriceForAnalyze(
				ctx,
				newStockBrandDailyPriceForAnalyzeFromGatewayPrices(gatewayPrices, now),
			); err != nil {
			return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze error")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	// 権利落ち(分割・併合)が発生した銘柄はfor_analyzeの当該銘柄の全期間を
	// J-Quantsの調整済みOHLCVで再取得・上書きする。外部APIを呼ぶためトランザクションの外で行う。
	if err := si.applyCorporateActionsForAnalyze(ctx, gatewayPrices, now); err != nil {
		return errors.Wrap(err, "applyCorporateActionsForAnalyze error")
	}

	return nil
}

// newStockBrandDailyPricesFromGateway - 全銘柄の一日の日足スライス（素値）を作成する
func (si *stockBrandsDailyStockPriceInteractorImpl) newStockBrandDailyPricesFromGateway(currentBrandsMap map[string]*models.StockBrand, stockPrices []*gateway.StockPrice, now time.Time) []*models.StockBrandDailyPrice {
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

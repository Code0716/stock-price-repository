package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

func (ii *indexInteractorImpl) CreateNikkeiAndDjiHistoricalData(ctx context.Context, t time.Time) error {
	// 日経平均の取得
	nikkeiDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolNikkei,
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange10Y,
	)
	if err != nil {
		return errors.Wrap(err, "CreateNikkeiAndDjiHistoricalData")
	}
	// NYダウの取得
	djiDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolDji,
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange10Y,
	)
	if err != nil {
		return errors.Wrap(err, "CreateNikkeiAndDjiHistoricalData")
	}
	// TOPIX連動ETF (1306.T) の取得
	// NEXT FUNDS TOPIX連動ETF (1306.T) を TOPIX 代理として使用（Yahoo に TOPIX 指数本体が無いため）
	topixDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolTopixETF,
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange10Y,
	)
	if err != nil {
		return errors.Wrap(err, "CreateNikkeiAndDjiHistoricalData")
	}

	err = ii.tx.DoInTx(ctx, func(ctx context.Context) error {
		if err := ii.nikkeiRepository.CreateNikkeiStockAverageDailyPrices(ctx, ii.apiResponseToModel(nikkeiDailyPrices, t)); err != nil {
			return errors.Wrap(err, "")
		}
		if err := ii.djiRepository.CreateDjiStockAverageDailyPrices(ctx, ii.apiResponseToModel(djiDailyPrices, t)); err != nil {
			return errors.Wrap(err, "")
		}
		if err := ii.topixRepository.CreateTopixDailyPrices(ctx, ii.apiResponseToModel(topixDailyPrices, t)); err != nil {
			return errors.Wrap(err, "")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

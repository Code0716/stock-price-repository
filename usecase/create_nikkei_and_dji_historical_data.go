//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/pkg/errors"
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
	// NYダウ兵器の取得
	djiDailyPrices, err := ii.stockAPIClient.GetIndexPriceChart(
		ctx,
		gateway.StockAPISymbolDji,
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
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

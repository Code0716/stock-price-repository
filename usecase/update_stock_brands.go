package usecase

import (
	"context"
	"log"
	"time"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
)

func (si *stockBrandInteractorImpl) UpdateStockBrands(ctx context.Context, now time.Time) error {
	// 取得
	stockBrandsInfo, err := si.stockAPIClient.GetStockBrands(ctx)
	if err != nil {
		return errors.Wrap(err, "UpdateStockBrands error")
	}

	if len(stockBrandsInfo) == 0 {
		log.Print("stockBrandInteractorImpl.UpdateStockBrands stockBrandsInfo 0")
		return nil
	}

	// 誤差なくすためにミリ秒を削除する。
	truncatedTime := now.Truncate(time.Second)
	stockBrands := make([]*models.StockBrand, 0, len(stockBrandsInfo))
	for _, v := range stockBrandsInfo {
		stockBrands = append(
			stockBrands,
			models.NewStockBrand(
				v.Symbol,
				v.CompanyName,
				v.MarketCode,
				v.MarketCodeName,
				v.Sector33Code,
				v.Sector33CodeName,
				v.Sector17Code,
				v.Sector17CodeName,
				truncatedTime,
				truncatedTime,
			))
	}

	err = si.tx.DoInTx(ctx, func(ctx context.Context) error {
		// 銘柄を取得
		currentBrands, err := si.stockBrandRepository.FindAll(ctx)
		if err != nil {
			return errors.Wrap(err, "stockBrandRepository.FindAll error")
		}

		for _, v := range stockBrands {
			// 既に存在する銘柄は更新
			for _, current := range currentBrands {
				if v.TickerSymbol == current.TickerSymbol {
					v.ID = current.ID
				}
			}
		}
		// 銘柄を保存
		if err := si.stockBrandRepository.UpsertStockBrands(ctx, stockBrands); err != nil {
			return errors.Wrap(err, "stockBrandRepository.UpsertStockBrands error")
		}

		// 上場廃止銘柄の取得 upsertされてなかったら上場廃止と判断する
		deleteIDs, err := si.stockBrandRepository.FindDelistingStockBrandsFromUpdateTime(ctx, truncatedTime)
		if err != nil {
			return errors.Wrap(err, "stockBrandRepository.FindDelistingStockBrandsFromUpdateTime error")
		}

		if len(deleteIDs) != 0 {
			// 上場廃止銘柄の日足削除
			if err := si.stockBrandsDailyPriceRepository.DeleteByIDs(ctx, deleteIDs); err != nil {
				return errors.Wrap(err, "stockBrandsDailyPriceRepository.DeleteDelisting error")
			}

			if err := si.analyzeStockBrandPriceHistoryRepository.DeleteByStockBrandIDs(ctx, deleteIDs); err != nil {
				return errors.Wrap(err, "analyzeStockBrandPriceHistoryRepository.DeleteByStockBrandIDs error")
			}

			// 上場廃止銘柄の削除 これが最後
			if err := si.stockBrandRepository.DeleteDelistingStockBrands(ctx, deleteIDs); err != nil {
				return errors.Wrap(err, "stockBrandRepository.DeleteDelistingStockBrands error")
			}
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	return nil
}

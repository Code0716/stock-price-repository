//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
)

type stockBrandsDailyStockPriceInteractorImpl struct {
	tx                                        repositories.Transaction
	stockBrandRepository                      repositories.StockBrandRepository
	stockBrandsDailyStockPriceRepository      repositories.StockBrandsDailyPriceRepository
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
	stockAPIClient                            gateway.StockAPIClient
	redisClient                               *redis.Client
	slackAPIClient                            gateway.SlackAPIClient
}

type StockBrandsDailyPriceInteractor interface {
	CreateDailyStockPrice(ctx context.Context, now time.Time) error
	CreateHistoricalDailyStockPrices(ctx context.Context, now time.Time) error
	AdjustHistoricalDataForStockSplit(ctx context.Context, symbol string, splitRatio decimal.Decimal, effectiveDate time.Time, dryRun bool) error
	GetDailyStockPrices(ctx context.Context, symbol string, from, to *time.Time) ([]*models.StockBrandDailyPrice, error)
	GetDailyStockPricesWithOrder(ctx context.Context, symbol string, from, to *time.Time, order *models.SortOrder) ([]*models.StockBrandDailyPrice, error)
}

func NewStockBrandsDailyPriceInteractor(
	tx repositories.Transaction,
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	stockAPIClient gateway.StockAPIClient,
	redisClient *redis.Client,
	slackAPIClient gateway.SlackAPIClient,
) StockBrandsDailyPriceInteractor {
	return &stockBrandsDailyStockPriceInteractorImpl{
		tx,
		stockBrandRepository,
		stockBrandsDailyStockPriceRepository,
		stockBrandsDailyPriceForAnalyzeRepository,
		stockAPIClient,
		redisClient,
		slackAPIClient,
	}
}

// AdjustHistoricalDataForStockSplit adjusts historical data for stock splits.
func (s *stockBrandsDailyStockPriceInteractorImpl) AdjustHistoricalDataForStockSplit(
	ctx context.Context,
	symbol string,
	splitRatio decimal.Decimal,
	effectiveDate time.Time,
	dryRun bool,
) error {
	// 1. Fetch daily stock prices up to the day before the effective date
	targetDateTo := effectiveDate.AddDate(0, 0, -1)
	filter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateTo:       &targetDateTo,
		DateOrder:    util.ToPtrGenerics(models.SortOrderAsc),
	}

	dailyPrices, err := s.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	if len(dailyPrices) == 0 {
		log.Printf("No daily prices found for %s before %v", symbol, effectiveDate)
		return nil
	}

	log.Printf("Found %d records for %s to adjust", len(dailyPrices), symbol)

	// 2. Adjust prices and volume
	var adjustedPrices []*models.StockBrandDailyPrice
	// var adjustedPricesForAnalyze []*models.StockBrandDailyPriceForAnalyze // If needed later

	for _, price := range dailyPrices {
		originalOpen := price.Open
		originalClose := price.Close
		originalVolume := price.Volume

		// Adjust prices: divide by split ratio
		price.Open = price.Open.Div(splitRatio)
		price.High = price.High.Div(splitRatio)
		price.Low = price.Low.Div(splitRatio)
		price.Close = price.Close.Div(splitRatio)
		price.Adjclose = price.Adjclose.Div(splitRatio)

		// Adjust volume: multiply by split ratio
		price.Volume = int64(decimal.NewFromInt(price.Volume).Mul(splitRatio).IntPart())

		if dryRun {
			log.Printf(
				"DryRun: Date=%s, Open: %s -> %s, Close: %s -> %s, Volume: %d -> %d",
				price.Date.Format("2006-01-02"),
				originalOpen, price.Open,
				originalClose, price.Close,
				originalVolume, price.Volume,
			)
		}

		adjustedPrices = append(adjustedPrices, price)
		// Assuming we also need to update data for analyze if the repository exists and models match
		// For now, let's focus on the main daily prices table
	}

	if dryRun {
		log.Println("DryRun completed. No changes saved.")
		return nil
	}

	// 3. Save changes in transaction
	err = s.tx.DoInTx(ctx, func(ctx context.Context) error {
		// Bulk update or upsert? Repository CreateStockBrandDailyPrice uses upsert logic
		if err := s.stockBrandsDailyStockPriceRepository.CreateStockBrandDailyPrice(ctx, adjustedPrices); err != nil {
			return errors.Wrap(err, "CreateStockBrandDailyPrice error")
		}
		
		// TODO: Also update StockBrandsDailyPriceForAnalyze if required
		
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	log.Printf("Successfully adjusted %d records for %s", len(adjustedPrices), symbol)
	return nil
}

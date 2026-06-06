//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

// minTechnicalIndicatorDays ADX のウォームアップ（2*period-1 = 27）を考慮した最小日数。
const minTechnicalIndicatorDays = 40

const (
	atrPeriod   = 14
	stochK      = 14
	stochD      = 3
	adxPeriod   = 14
	vwapPeriod  = 14
)

type technicalIndicatorsInteractorImpl struct {
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
}

type TechnicalIndicatorsInteractor interface {
	// GetTechnicalIndicators 指定銘柄の期間テクニカル指標時系列を返す。
	GetTechnicalIndicators(ctx context.Context, symbol string, from, to *time.Time) (*models.TechnicalIndicators, error)
}

func NewTechnicalIndicatorsInteractor(
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
) TechnicalIndicatorsInteractor {
	return &technicalIndicatorsInteractorImpl{
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
	}
}

func (t *technicalIndicatorsInteractorImpl) GetTechnicalIndicators(ctx context.Context, symbol string, from, to *time.Time) (*models.TechnicalIndicators, error) {
	order := models.SortOrderAsc
	prices, err := t.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
		DateOrder:    &order,
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	result := &models.TechnicalIndicators{
		Symbol:      symbol,
		TradingDays: len(prices),
	}
	if len(prices) > 0 {
		result.From = prices[0].Date.Format(util.DateLayout)
		result.To = prices[len(prices)-1].Date.Format(util.DateLayout)
	}

	if len(prices) < minTechnicalIndicatorDays {
		return result, nil
	}

	atr := domain_service.CalculateATR(prices, atrPeriod)
	stoch := domain_service.CalculateStochastics(prices, stochK, stochD)
	adx := domain_service.CalculateADX(prices, adxPeriod)
	obv := domain_service.CalculateOBV(prices)
	vwap := domain_service.CalculateRollingVWAP(prices, vwapPeriod)
	result.Points = buildTechnicalIndicatorPoints(prices, atr, stoch, adx, obv, vwap)

	return result, nil
}

func buildTechnicalIndicatorPoints(
	prices []*models.StockBrandDailyPrice,
	atr []decimal.Decimal,
	stoch []domain_service.StochasticsResult,
	adx []domain_service.ADXResult,
	obv []decimal.Decimal,
	vwap []decimal.Decimal,
) []models.TechnicalIndicatorPoint {
	points := make([]models.TechnicalIndicatorPoint, len(prices))
	for i, p := range prices {
		c := p.Close
		pt := models.TechnicalIndicatorPoint{
			Date:  p.Date.Format(util.DateLayout),
			Close: &c,
		}
		if atr != nil && !atr[i].IsZero() {
			v := atr[i]
			pt.ATR = &v
		}
		if stoch != nil && !stoch[i].K.IsZero() {
			k := stoch[i].K
			d := stoch[i].D
			pt.StochK = &k
			if !d.IsZero() {
				pt.StochD = &d
			}
		}
		if adx != nil {
			if !adx[i].PlusDI.IsZero() {
				v := adx[i].PlusDI
				pt.PlusDI = &v
				v2 := adx[i].MinusDI
				pt.MinusDI = &v2
			}
			if !adx[i].ADX.IsZero() {
				v := adx[i].ADX
				pt.ADX = &v
			}
		}
		if obv != nil {
			// OBV は index 0 も 0（基準値）なので常に非 nil で返す
			v := obv[i]
			pt.OBV = &v
		}
		if vwap != nil && !vwap[i].IsZero() {
			v := vwap[i]
			pt.VWAP = &v
		}
		points[i] = pt
	}
	return points
}

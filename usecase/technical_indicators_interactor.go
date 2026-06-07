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
	atrPeriod            = 14
	stochK               = 14
	stochD               = 3
	adxPeriod            = 14
	vwapPeriod           = 14
	ichimokuConv         = 9
	ichimokuBase         = 26
	ichimokuSpanB        = 52
	ichimokuDisplacement = 26
	srLookback           = 3
)

var srTolerance = decimal.NewFromFloat(0.015)

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
		Symbol:                  symbol,
		TradingDays:             len(prices),
		FuturePoints:            []models.TechnicalIndicatorPoint{},
		SupportResistanceLevels: []models.SupportResistanceLevel{},
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
	ich := domain_service.CalculateIchimoku(prices, ichimokuConv, ichimokuBase, ichimokuSpanB)
	levels := domain_service.CalculateSupportResistance(prices, srLookback, srTolerance)

	result.Points = buildTechnicalIndicatorPoints(prices, atr, stoch, adx, obv, vwap, ich)
	result.FuturePoints = buildFuturePoints(prices, ich)
	result.SupportResistanceLevels = buildSupportResistanceLevels(levels, prices)

	return result, nil
}

func buildTechnicalIndicatorPoints(
	prices []*models.StockBrandDailyPrice,
	atr []decimal.Decimal,
	stoch []domain_service.StochasticsResult,
	adx []domain_service.ADXResult,
	obv []decimal.Decimal,
	vwap []decimal.Decimal,
	ich []domain_service.IchimokuResult,
) []models.TechnicalIndicatorPoint {
	n := len(prices)
	points := make([]models.TechnicalIndicatorPoint, n)
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
		applyIchimokuShift(&pt, prices, ich, i, n)
		points[i] = pt
	}
	return points
}

// applyIchimokuShift 一目均衡表のシフトを適用して pt に書き込む。
// - Tenkan/Kijun: 当日値をそのまま
// - SenkouA/B: Points[i] には ich[i-disp] の生値（+26日先に投影されるため）
// - Chikou: Points[i-disp] に prices[i].Close を書く（front で当日として描画すると -26日に見える）
func applyIchimokuShift(pt *models.TechnicalIndicatorPoint, prices []*models.StockBrandDailyPrice, ich []domain_service.IchimokuResult, i, n int) {
	if ich == nil {
		return
	}
	disp := ichimokuDisplacement
	if !ich[i].Tenkan.IsZero() {
		v := ich[i].Tenkan
		pt.Tenkan = &v
	}
	if !ich[i].Kijun.IsZero() {
		v := ich[i].Kijun
		pt.Kijun = &v
	}
	// SenkouA/B を当日 i に表示するには ich[i-disp] の生値を使う
	if i >= disp {
		if !ich[i-disp].SenkouA.IsZero() {
			v := ich[i-disp].SenkouA
			pt.SenkouA = &v
		}
		if !ich[i-disp].SenkouB.IsZero() {
			v := ich[i-disp].SenkouB
			pt.SenkouB = &v
		}
	}
	// Chikou: i+disp < n の日は prices[i+disp].Close を当日ポイントに置く。
	// front は Points[i].date に Chikou 値を描画 → 26日先の終値が 26日前の位置に表示される。
	if i+disp < n {
		v := prices[i+disp].Close
		pt.Chikou = &v
	}
}

func buildFuturePoints(prices []*models.StockBrandDailyPrice, ich []domain_service.IchimokuResult) []models.TechnicalIndicatorPoint {
	if ich == nil {
		return []models.TechnicalIndicatorPoint{}
	}
	n := len(prices)
	if n == 0 {
		return []models.TechnicalIndicatorPoint{}
	}

	disp := ichimokuDisplacement
	lastDate := prices[n-1].Date
	futureDates := nextBusinessDays(lastDate, disp)
	futurePoints := make([]models.TechnicalIndicatorPoint, disp)

	for j := range disp {
		pt := models.TechnicalIndicatorPoint{
			Date: futureDates[j].Format(util.DateLayout),
		}
		// j 番目の未来点のスパン = ich[n-disp+j] の生値（+disp シフト後の値）
		srcIdx := n - disp + j
		if srcIdx >= 0 && srcIdx < n {
			if !ich[srcIdx].SenkouA.IsZero() {
				v := ich[srcIdx].SenkouA
				pt.SenkouA = &v
			}
			if !ich[srcIdx].SenkouB.IsZero() {
				v := ich[srcIdx].SenkouB
				pt.SenkouB = &v
			}
		}
		futurePoints[j] = pt
	}
	return futurePoints
}

// nextBusinessDays lastDate の翌日から count 本の営業日（土日スキップ）を返す。
func nextBusinessDays(lastDate time.Time, count int) []time.Time {
	dates := make([]time.Time, 0, count)
	cur := lastDate
	for len(dates) < count {
		cur = cur.AddDate(0, 0, 1)
		if cur.Weekday() == time.Saturday || cur.Weekday() == time.Sunday {
			continue
		}
		dates = append(dates, cur)
	}
	return dates
}

func buildSupportResistanceLevels(levels []domain_service.SwingLevel, prices []*models.StockBrandDailyPrice) []models.SupportResistanceLevel {
	if len(levels) == 0 || len(prices) == 0 {
		return []models.SupportResistanceLevel{}
	}
	lastClose := prices[len(prices)-1].Close
	result := make([]models.SupportResistanceLevel, 0, len(levels))
	for _, lv := range levels {
		price := lv.Price
		kind := "resistance"
		if price.LessThanOrEqual(lastClose) {
			kind = "support"
		}
		lastDate := ""
		if lv.LastIdx >= 0 && lv.LastIdx < len(prices) {
			lastDate = prices[lv.LastIdx].Date.Format(util.DateLayout)
		}
		result = append(result, models.SupportResistanceLevel{
			Price:    &price,
			Kind:     kind,
			Touches:  lv.Touches,
			LastDate: lastDate,
		})
	}
	return result
}

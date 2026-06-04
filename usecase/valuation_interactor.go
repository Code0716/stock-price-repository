//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/util"
)

const (
	// valuationFinStatementsLimit 財務データの取得件数。直近通期(FY)EPSを探すのに十分な数。
	valuationFinStatementsLimit = 12
	// typeOfCurrentPeriodFY 通期（年次）決算を示す値。EPS算出には通期のみ使用する。
	typeOfCurrentPeriodFY = "FY"
)

type valuationInteractorImpl struct {
	finStatementRepository               repositories.FinStatementRepository
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
}

// ValuationInteractor 評価指標（PER/PBR/ROE/予想PER）算出インターフェース。
type ValuationInteractor interface {
	GetValuation(ctx context.Context, symbol string) (*models.Valuation, error)
}

func NewValuationInteractor(
	finStatementRepository repositories.FinStatementRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
) ValuationInteractor {
	return &valuationInteractorImpl{
		finStatementRepository:               finStatementRepository,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
	}
}

func (v *valuationInteractorImpl) GetValuation(ctx context.Context, symbol string) (*models.Valuation, error) {
	// 最新終値を取得（エラーは500で伝播）
	latestPrice, err := v.stockBrandsDailyStockPriceRepository.GetLatestPriceBySymbol(ctx, symbol)
	if err != nil {
		return nil, errors.Wrap(err, "GetLatestPriceBySymbol error")
	}

	result := &models.Valuation{Symbol: symbol}
	if latestPrice != nil {
		result.Close = &latestPrice.Close
		result.PriceDate = latestPrice.Date.Format(util.DateLayout)
	}

	// 財務データを取得（空の場合は各指標 nil で返す）
	statements, err := v.finStatementRepository.FindBySymbol(ctx, &models.FinStatementFilter{
		TickerSymbol: symbol,
		Limit:        valuationFinStatementsLimit,
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindBySymbol error")
	}
	if len(statements) == 0 {
		return result, nil
	}

	// 財務12件（disclosed_date 降順）から各値を抽出
	trailingEPS, forecastEPS, bps, fiscalPeriod := extractFinancialValues(statements)
	result.TrailingEPS = trailingEPS
	result.ForecastEPS = forecastEPS
	result.BPS = bps
	result.FiscalPeriod = fiscalPeriod

	// 指標を算出（終値が取れている場合のみ）
	if result.Close != nil {
		close := *result.Close
		computed := computeValuation(close, trailingEPS, forecastEPS, bps)
		result.PER = computed.PER
		result.ForwardPER = computed.ForwardPER
		result.PBR = computed.PBR
		result.ROE = computed.ROE
	}

	return result, nil
}

// extractFinancialValues 財務スライス（disclosed_date 降順）から実績EPS/予想EPS/BPSを抽出する。
// 実績EPS は通期(FY)決算のみ使用する（四半期EPSは期中累計で年換算でないため除外）。
func extractFinancialValues(statements []*models.FinStatement) (trailingEPS *decimal.Decimal, forecastEPS *decimal.Decimal, bps *decimal.Decimal, fiscalPeriod string) {
	for _, s := range statements {
		if trailingEPS == nil && s.TypeOfCurrentPeriod == typeOfCurrentPeriodFY && s.EarningsPerShare != nil {
			trailingEPS = s.EarningsPerShare
			if s.FiscalYearEnd != nil {
				fiscalPeriod = s.FiscalYearEnd.Format("2006-01")
			}
		}
		if forecastEPS == nil && s.ForecastEPS != nil {
			forecastEPS = s.ForecastEPS
		}
		if bps == nil && s.BookValuePerShare != nil {
			bps = s.BookValuePerShare
		}
		if trailingEPS != nil && forecastEPS != nil && bps != nil {
			break
		}
	}
	return
}

// valuationMetrics computeValuation の戻り値。
type valuationMetrics struct {
	PER        *decimal.Decimal
	ForwardPER *decimal.Decimal
	PBR        *decimal.Decimal
	ROE        *decimal.Decimal
}

// computeValuation 終値と財務値から評価指標を算出する純粋関数。
// 割る数が nil/0/負 のときは該当指標を nil とする。
// ただし ROE は負EPS（赤字）でも算出する（負ROEは意味があるため）。
func computeValuation(close decimal.Decimal, trailingEPS, forecastEPS, bps *decimal.Decimal) valuationMetrics {
	m := valuationMetrics{}

	// PER = close / trailingEPS（EPS > 0 のときのみ）
	if trailingEPS != nil && trailingEPS.IsPositive() {
		v := close.Div(*trailingEPS)
		m.PER = &v
	}

	// ForwardPER = close / forecastEPS（forecastEPS > 0 のときのみ）
	if forecastEPS != nil && forecastEPS.IsPositive() {
		v := close.Div(*forecastEPS)
		m.ForwardPER = &v
	}

	// PBR = close / bps（bps > 0 のときのみ）
	if bps != nil && bps.IsPositive() {
		v := close.Div(*bps)
		m.PBR = &v
	}

	// ROE = trailingEPS / bps（bps > 0 のとき。負EPSも算出する）
	if trailingEPS != nil && bps != nil && bps.IsPositive() {
		v := trailingEPS.Div(*bps)
		m.ROE = &v
	}

	return m
}

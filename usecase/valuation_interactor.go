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

// ValuationInteractor 評価指標（PER/PBR/ROE/予想PER/予想配当利回り）算出インターフェース。
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
	fv := extractFinancialValues(statements)
	result.TrailingEPS = fv.trailingEPS
	result.ForecastEPS = fv.forecastEPS
	result.BPS = fv.bps
	result.ForecastDividendPerShareAnnual = fv.forecastDPS
	result.FiscalPeriod = fv.fiscalPeriod

	// 指標を算出（終値が取れている場合のみ）
	if result.Close != nil {
		close := *result.Close
		computed := computeValuation(close, fv.trailingEPS, fv.forecastEPS, fv.bps, fv.forecastDPS)
		result.PER = computed.PER
		result.ForwardPER = computed.ForwardPER
		result.PBR = computed.PBR
		result.ROE = computed.ROE
		result.ForecastDividendYield = computed.ForecastDividendYield
	}

	return result, nil
}

// financialValues extractFinancialValues の戻り値をまとめた構造体。
type financialValues struct {
	trailingEPS  *decimal.Decimal
	forecastEPS  *decimal.Decimal
	bps          *decimal.Decimal
	forecastDPS  *decimal.Decimal
	fiscalPeriod string
}

// extractFinancialValues 財務スライス（disclosed_date 降順）から実績EPS/予想EPS/BPS/予想DPSを抽出する。
// 実績EPS は通期(FY)決算のみ使用する（四半期EPSは期中累計で年換算でないため除外）。
func extractFinancialValues(statements []*models.FinStatement) financialValues {
	var v financialValues
	for _, s := range statements {
		extractTrailingEPS(s, &v)
		if v.forecastEPS == nil && s.ForecastEPS != nil {
			v.forecastEPS = s.ForecastEPS
		}
		if v.bps == nil && s.BookValuePerShare != nil {
			v.bps = s.BookValuePerShare
		}
		if v.forecastDPS == nil && s.ForecastDividendPerShareAnnual != nil {
			v.forecastDPS = s.ForecastDividendPerShareAnnual
		}
		if v.trailingEPS != nil && v.forecastEPS != nil && v.bps != nil && v.forecastDPS != nil {
			break
		}
	}
	return v
}

func extractTrailingEPS(s *models.FinStatement, v *financialValues) {
	if v.trailingEPS != nil || s.TypeOfCurrentPeriod != typeOfCurrentPeriodFY || s.EarningsPerShare == nil {
		return
	}
	v.trailingEPS = s.EarningsPerShare
	if s.FiscalYearEnd != nil {
		v.fiscalPeriod = s.FiscalYearEnd.Format("2006-01")
	}
}

// valuationMetrics computeValuation の戻り値。
type valuationMetrics struct {
	PER                   *decimal.Decimal
	ForwardPER            *decimal.Decimal
	PBR                   *decimal.Decimal
	ROE                   *decimal.Decimal
	ForecastDividendYield *decimal.Decimal
}

// computeValuation 終値と財務値から評価指標を算出する純粋関数。
// 割る数が nil/0/負 のときは該当指標を nil とする。
// ただし ROE は負EPS（赤字）でも算出する（負ROEは意味があるため）。
func computeValuation(close decimal.Decimal, trailingEPS, forecastEPS, bps, forecastDPS *decimal.Decimal) valuationMetrics {
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

	// ForecastDividendYield = forecastDPS / close（close > 0 のときのみ）
	if forecastDPS != nil && close.IsPositive() {
		v := forecastDPS.Div(close)
		m.ForecastDividendYield = &v
	}

	return m
}

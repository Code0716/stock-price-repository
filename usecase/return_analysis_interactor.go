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

// minReturnAnalysisDays リターン分析に必要な最小データ点数（リターンを1本以上得るため2日必要）。
const minReturnAnalysisDays = 2

type returnAnalysisInteractorImpl struct {
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
	nikkeiRepository                     repositories.NikkeiRepository
	topixRepository                      repositories.TopixRepository
}

type ReturnAnalysisInteractor interface {
	// GetReturnAnalysis 指定銘柄の期間リターン・リスク指標・対ベンチマーク指標を算出する。
	// benchmark は "nikkei"（デフォルト）または "topix"。
	GetReturnAnalysis(ctx context.Context, symbol string, from, to *time.Time, benchmark string) (*models.ReturnAnalysis, error)
}

func NewReturnAnalysisInteractor(
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	nikkeiRepository repositories.NikkeiRepository,
	topixRepository repositories.TopixRepository,
) ReturnAnalysisInteractor {
	return &returnAnalysisInteractorImpl{
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		nikkeiRepository:                     nikkeiRepository,
		topixRepository:                      topixRepository,
	}
}

func (r *returnAnalysisInteractorImpl) GetReturnAnalysis(ctx context.Context, symbol string, from, to *time.Time, benchmark string) (*models.ReturnAnalysis, error) {
	// 銘柄日足（時系列・昇順）
	order := models.SortOrderAsc
	stockPrices, err := r.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
		TickerSymbol: symbol,
		DateFrom:     from,
		DateTo:       to,
		DateOrder:    &order,
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListDailyPricesBySymbol error")
	}

	// ベンチマーク日足の取得（nikkei / topix 切替）
	var benchPrices models.IndexStockAverageDailyPrices
	var benchmarkLabel string
	switch benchmark {
	case models.BenchmarkTopix:
		benchPrices, err = r.topixRepository.ListTopixDailyPrices(ctx, from, to)
		if err != nil {
			return nil, errors.Wrap(err, "ListTopixDailyPrices error")
		}
		benchmarkLabel = models.BenchmarkTopix
	default:
		benchPrices, err = r.nikkeiRepository.ListNikkeiStockAverageDailyPrices(ctx, from, to)
		if err != nil {
			return nil, errors.Wrap(err, "ListNikkeiStockAverageDailyPrices error")
		}
		benchmarkLabel = models.BenchmarkNikkei
	}

	// 銘柄とベンチマークを日付で内部結合（両方に存在する日のみ採用）
	stockAdjClose, benchAdjClose, dates := alignByDate(stockPrices, benchPrices)

	result := &models.ReturnAnalysis{
		Symbol:      symbol,
		Benchmark:   benchmarkLabel,
		TradingDays: len(dates),
	}
	if len(dates) > 0 {
		result.From = dates[0].Format(util.DateLayout)
		result.To = dates[len(dates)-1].Format(util.DateLayout)
	}

	// データ不足時はゼロ値の指標で返す（フロントは TradingDays で判定可能）
	if len(stockAdjClose) < minReturnAnalysisDays {
		return result, nil
	}

	stockDailyReturns := domain_service.DailyReturns(stockAdjClose)
	benchDailyReturns := domain_service.DailyReturns(benchAdjClose)

	cumulativeReturn := domain_service.CumulativeReturn(stockAdjClose)
	benchmarkReturn := domain_service.CumulativeReturn(benchAdjClose)
	annualizedReturn := domain_service.AnnualizedReturn(cumulativeReturn, len(stockDailyReturns))
	annualizedVol := domain_service.AnnualizedVolatility(stockDailyReturns)
	maxDrawdown := domain_service.MaxDrawdown(stockAdjClose)
	downsideDeviation := domain_service.AnnualizedDownsideDeviation(stockDailyReturns, decimal.Zero)

	// リスクフリーレートは日本の超低金利を踏まえ 0 とする
	riskFreeRate := decimal.Zero

	result.CumulativeReturn = cumulativeReturn
	result.AnnualizedReturn = annualizedReturn
	result.AnnualizedVolatility = annualizedVol
	result.MaxDrawdown = maxDrawdown
	result.SharpeRatio = domain_service.SharpeRatio(annualizedReturn, annualizedVol, riskFreeRate)
	result.SortinoRatio = domain_service.SortinoRatio(annualizedReturn, downsideDeviation, riskFreeRate)
	result.CalmarRatio = domain_service.CalmarRatio(annualizedReturn, maxDrawdown)
	result.Beta = domain_service.Beta(stockDailyReturns, benchDailyReturns)
	result.Correlation = domain_service.Correlation(stockDailyReturns, benchDailyReturns)
	result.BenchmarkReturn = benchmarkReturn
	result.ExcessReturn = domain_service.ExcessReturn(cumulativeReturn, benchmarkReturn)

	return result, nil
}

// alignByDate 銘柄日足とベンチマーク日足を日付（日単位）で内部結合し、
// 両方に存在する日の adjClose 系列（昇順）と対応する日付を返す。
// stockPrices は昇順で渡される前提。
func alignByDate(
	stockPrices []*models.StockBrandDailyPrice,
	benchPrices models.IndexStockAverageDailyPrices,
) (stockAdjClose, benchAdjClose []decimal.Decimal, dates []time.Time) {
	benchByDate := make(map[string]decimal.Decimal, len(benchPrices))
	for _, b := range benchPrices {
		benchByDate[b.Date.Format(util.DateLayout)] = b.Adjclose
	}

	for _, s := range stockPrices {
		key := s.Date.Format(util.DateLayout)
		bench, ok := benchByDate[key]
		if !ok {
			continue
		}
		stockAdjClose = append(stockAdjClose, s.Adjclose)
		benchAdjClose = append(benchAdjClose, bench)
		dates = append(dates, s.Date)
	}
	return stockAdjClose, benchAdjClose, dates
}

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

const (
	strategyRankingRedisKey = "strategy_ranking:v1"
	strategyRankingUniverse = "main_markets"
	// strategyRankingMinDays は指標ウォームアップに必要な最低日数（backtest_interactor.go と同値）。
	strategyRankingMinDays = 80
)

// strategyAcc 1戦略の集計アキュムレータ。
type strategyAcc struct {
	stockCount    int
	tradedStocks  int
	positiveCount int
	sumTotalReturn decimal.Decimal
	sumWinRate     decimal.Decimal
	sumPF          decimal.Decimal
	totalTrades    int
	bestCount      int
}

// accumulateResults 日足と exitParams から各戦略の結果を accs に集計する。
func accumulateResults(prices []*models.StockBrandDailyPrice, exitParams domain_service.ExitParams, accs map[string]*strategyAcc) {
	results := make(map[string]models.BacktestResult, len(domain_service.StrategyOrder))
	for _, s := range domain_service.StrategyOrder {
		signals := domain_service.EntrySignalsByStrategy(s, prices)
		results[s] = domain_service.RunBacktest(prices, signals, exitParams)
	}
	for _, s := range domain_service.StrategyOrder {
		res := results[s]
		a := accs[s]
		a.stockCount++
		a.sumTotalReturn = a.sumTotalReturn.Add(res.TotalReturn)
		if res.TotalReturn.IsPositive() {
			a.positiveCount++
		}
		a.totalTrades += res.Trades
		if res.Trades > 0 {
			a.tradedStocks++
			a.sumWinRate = a.sumWinRate.Add(res.WinRate)
			a.sumPF = a.sumPF.Add(res.ProfitFactor)
		}
	}
	// この銘柄で最高 TotalReturn の戦略の bestCount を加算する
	bestStrategy := ""
	bestReturn := decimal.NewFromInt(-999)
	for _, s := range domain_service.StrategyOrder {
		if results[s].Trades > 0 && results[s].TotalReturn.GreaterThan(bestReturn) {
			bestReturn = results[s].TotalReturn
			bestStrategy = s
		}
	}
	if bestStrategy != "" {
		accs[bestStrategy].bestCount++
	}
}

type strategyRankingInteractorImpl struct {
	stockBrandRepository                 repositories.StockBrandRepository
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
	redisClient                          *redis.Client
}

// StrategyRankingInteractor 全銘柄横断バックテスト集計インターフェース。
type StrategyRankingInteractor interface {
	// ComputeAndSaveStrategyRanking 全主要市場銘柄を全戦略でバックテストし、集計を Redis に保存する。
	// years: 直近N年を対象期間とする。処理した銘柄数を返す。
	ComputeAndSaveStrategyRanking(ctx context.Context, params models.BacktestParams, years int) (int, error)
	// GetStrategyRanking Redis から集計を返す。未計算なら Computed=false の空の StrategyRanking を返す。
	GetStrategyRanking(ctx context.Context) (*models.StrategyRanking, error)
}

func NewStrategyRankingInteractor(
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	redisClient *redis.Client,
) StrategyRankingInteractor {
	return &strategyRankingInteractorImpl{
		stockBrandRepository:                 stockBrandRepository,
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		redisClient:                          redisClient,
	}
}

func (r *strategyRankingInteractorImpl) GetStrategyRanking(ctx context.Context) (*models.StrategyRanking, error) {
	raw, err := r.redisClient.Get(ctx, strategyRankingRedisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &models.StrategyRanking{Computed: false, Items: []models.StrategyRankingItem{}}, nil
		}
		return nil, errors.Wrap(err, "redisClient.Get error")
	}
	var ranking models.StrategyRanking
	if err := json.Unmarshal([]byte(raw), &ranking); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal error")
	}
	return &ranking, nil
}

func (r *strategyRankingInteractorImpl) ComputeAndSaveStrategyRanking(ctx context.Context, params models.BacktestParams, years int) (int, error) {
	brands, err := r.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "FindAllMainMarkets error")
	}

	now := time.Now()
	from := now.AddDate(-years, 0, 0)
	asc := models.SortOrderAsc
	exitParams := domain_service.ExitParams{
		TakeProfit:  params.TakeProfit,
		StopLoss:    params.StopLoss,
		MaxHoldDays: params.MaxHoldDays,
	}

	// 戦略ごとのアキュムレータを初期化
	accs := make(map[string]*strategyAcc, len(domain_service.StrategyOrder))
	for _, s := range domain_service.StrategyOrder {
		accs[s] = &strategyAcc{}
	}

	processed := 0
	for i, brand := range brands {
		prices, err := r.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(ctx, models.ListDailyPricesBySymbolFilter{
			TickerSymbol: brand.TickerSymbol,
			DateFrom:     &from,
			DateTo:       &now,
			DateOrder:    &asc,
		})
		if err != nil {
			return processed, errors.Wrap(err, fmt.Sprintf("ListDailyPricesBySymbol error for %s", brand.TickerSymbol))
		}
		if len(prices) < strategyRankingMinDays {
			continue
		}
		accumulateResults(prices, exitParams, accs)
		processed++
		if (i+1)%100 == 0 {
			log.Printf("strategy ranking: processed %d/%d brands", i+1, len(brands))
		}
	}

	// StrategyRankingItem を組み立て
	items := make([]models.StrategyRankingItem, 0, len(domain_service.StrategyOrder))
	for _, s := range domain_service.StrategyOrder {
		a := accs[s]
		item := models.StrategyRankingItem{
			Strategy:    s,
			Label:       domain_service.StrategyLabels[s],
			StockCount:  a.stockCount,
			TradedStocks: a.tradedStocks,
			TotalTrades: a.totalTrades,
			BestCount:   a.bestCount,
		}
		if a.stockCount > 0 {
			item.AvgTotalReturn = a.sumTotalReturn.Div(decimal.NewFromInt(int64(a.stockCount)))
			item.PositiveRate = decimal.NewFromInt(int64(a.positiveCount)).Div(decimal.NewFromInt(int64(a.stockCount)))
		}
		if a.tradedStocks > 0 {
			item.AvgWinRate = a.sumWinRate.Div(decimal.NewFromInt(int64(a.tradedStocks)))
			item.AvgProfitFactor = a.sumPF.Div(decimal.NewFromInt(int64(a.tradedStocks)))
		}
		items = append(items, item)
	}

	// AvgTotalReturn 降順でソート
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].AvgTotalReturn.GreaterThan(items[j].AvgTotalReturn)
	})

	ranking := models.StrategyRanking{
		Computed:    true,
		ComputedAt:  now.Format(time.RFC3339),
		Universe:    strategyRankingUniverse,
		TotalStocks: len(brands),
		Params:      params,
		Items:       items,
	}

	b, err := json.Marshal(ranking)
	if err != nil {
		return processed, errors.Wrap(err, "json.Marshal error")
	}
	if err := r.redisClient.Set(ctx, strategyRankingRedisKey, string(b), 0).Err(); err != nil {
		return processed, errors.Wrap(err, "redisClient.Set error")
	}

	log.Printf("strategy ranking: completed. processed=%d/%d brands", processed, len(brands))
	return processed, nil
}

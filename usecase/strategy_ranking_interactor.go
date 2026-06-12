//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

const (
	strategyRankingRedisKey        = "strategy_ranking:v1"
	strategyRankingStocksKeyPrefix = "strategy_ranking:v1:stocks:"
	strategyRankingUniverse        = "main_markets"
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
	// stocks 銘柄別のドリルダウン結果リスト（ドリルダウン API 用）
	stocks []*models.StrategyStockResult
}

// newAccs 戦略ごとの集計アキュムレータを初期化して返す。
func newAccs() map[string]*strategyAcc {
	accs := make(map[string]*strategyAcc, len(domain_service.StrategyOrder))
	for _, s := range domain_service.StrategyOrder {
		accs[s] = &strategyAcc{}
	}
	return accs
}

// mergeAccs src の集計を dst に加算する（ワーカーローカル集計のマージ用）。
func mergeAccs(dst, src map[string]*strategyAcc) {
	for _, s := range domain_service.StrategyOrder {
		d, sa := dst[s], src[s]
		d.stockCount += sa.stockCount
		d.tradedStocks += sa.tradedStocks
		d.positiveCount += sa.positiveCount
		d.sumTotalReturn = d.sumTotalReturn.Add(sa.sumTotalReturn)
		d.sumWinRate = d.sumWinRate.Add(sa.sumWinRate)
		d.sumPF = d.sumPF.Add(sa.sumPF)
		d.totalTrades += sa.totalTrades
		d.bestCount += sa.bestCount
		d.stocks = append(d.stocks, sa.stocks...)
	}
}

// accumulateResults 日足と exitParams から各戦略の結果を accs に集計する。
// 集計用途のため Equity/TradeList を構築しない RunBacktestMetrics を使う。
func accumulateResults(brand *models.StockBrand, prices []*models.StockBrandDailyPrice, exitParams domain_service.ExitParams, accs map[string]*strategyAcc) {
	results := make(map[string]models.BacktestResult, len(domain_service.StrategyOrder))
	for _, s := range domain_service.StrategyOrder {
		signals := domain_service.EntrySignalsByStrategy(s, prices)
		// exitSignals は nil を渡して共通ルールのみ使用（ランキングバッチは挙動不変を優先）
		results[s] = domain_service.RunBacktestMetrics(prices, signals, nil, exitParams)
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
		// 銘柄別ドリルダウン用に結果を蓄積
		a.stocks = append(a.stocks, &models.StrategyStockResult{
			TickerSymbol: brand.TickerSymbol,
			Name:         brand.Name,
			TotalReturn:  res.TotalReturn,
			Trades:       res.Trades,
			WinRate:      res.WinRate,
			ProfitFactor: res.ProfitFactor,
			MaxDrawdown:  res.MaxDrawdown,
			PayoffRatio:  res.PayoffRatio,
			AvgHoldDays:  res.AvgHoldDays,
		})
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
	// years: 直近N年を対象期間とする。concurrency: ワーカー数（<=0 で NumCPU）。処理した銘柄数を返す。
	ComputeAndSaveStrategyRanking(ctx context.Context, params models.BacktestParams, years, concurrency int) (int, error)
	// GetStrategyRanking Redis から集計を返す。未計算なら Computed=false の空の StrategyRanking を返す。
	GetStrategyRanking(ctx context.Context) (*models.StrategyRanking, error)
	// GetStrategyRankingStocks Redis から戦略別の銘柄ドリルダウン結果を返す。
	// 未計算なら Computed=false の空の StrategyStocks を返す。Items は limit 件に切り、TotalCount は全件数。
	GetStrategyRankingStocks(ctx context.Context, strategy string, limit int) (*models.StrategyStocks, error)
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

func (r *strategyRankingInteractorImpl) GetStrategyRankingStocks(ctx context.Context, strategy string, limit int) (*models.StrategyStocks, error) {
	key := strategyRankingStocksKeyPrefix + strategy
	raw, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &models.StrategyStocks{
				Computed: false,
				Strategy: strategy,
				Items:    []*models.StrategyStockResult{},
			}, nil
		}
		return nil, errors.Wrap(err, "redisClient.Get error")
	}
	var stocks models.StrategyStocks
	if err := json.Unmarshal([]byte(raw), &stocks); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal error")
	}
	// TotalCount は全件数
	stocks.TotalCount = len(stocks.Items)
	// limit 件に切る
	if limit > 0 && len(stocks.Items) > limit {
		stocks.Items = stocks.Items[:limit]
	}
	return &stocks, nil
}

func (r *strategyRankingInteractorImpl) ComputeAndSaveStrategyRanking(ctx context.Context, params models.BacktestParams, years, concurrency int) (int, error) {
	brands, err := r.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "FindAllMainMarkets error")
	}

	now := time.Now()
	from := now.AddDate(-years, 0, 0)
	exitParams := domain_service.ExitParams{
		TakeProfit:  params.TakeProfit,
		StopLoss:    params.StopLoss,
		MaxHoldDays: params.MaxHoldDays,
	}

	accs, processed, err := r.runWorkers(ctx, brands, from, now, exitParams, concurrency)
	if err != nil {
		return processed, err
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

	// 戦略別銘柄ドリルダウンデータを Redis に保存
	computedAt := now.Format(time.RFC3339)
	for _, s := range domain_service.StrategyOrder {
		a := accs[s]
		// TotalReturn 降順にソート
		sort.SliceStable(a.stocks, func(i, j int) bool {
			return a.stocks[i].TotalReturn.GreaterThan(a.stocks[j].TotalReturn)
		})
		stocksPayload := models.StrategyStocks{
			Computed:   true,
			ComputedAt: computedAt,
			Strategy:   s,
			Label:      domain_service.StrategyLabels[s],
			TotalCount: len(a.stocks),
			Items:      a.stocks,
		}
		sb, err := json.Marshal(stocksPayload)
		if err != nil {
			return processed, errors.Wrap(err, "json.Marshal stocks error for "+s)
		}
		key := strategyRankingStocksKeyPrefix + s
		if err := r.redisClient.Set(ctx, key, string(sb), 0).Err(); err != nil {
			return processed, errors.Wrap(err, "redisClient.Set stocks error for "+s)
		}
	}

	log.Printf("strategy ranking: completed. processed=%d/%d brands", processed, len(brands))
	return processed, nil
}

// runWorkers 固定 concurrency 個のワーカーで全銘柄を並列にバックテストし、
// ワーカーローカルに集計してからマージした accs と処理銘柄数を返す。
// 各ワーカーは自分専用の accs にのみ書き込むためロック不要。decimal の総和は
// 順序非依存で厳密なので、結果は逐次版と完全一致する。
func (r *strategyRankingInteractorImpl) runWorkers(
	ctx context.Context,
	brands []*models.StockBrand,
	from, to time.Time,
	exitParams domain_service.ExitParams,
	concurrency int,
) (map[string]*strategyAcc, int, error) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	asc := models.SortOrderAsc

	workerAccs := make([]map[string]*strategyAcc, concurrency)
	for w := range workerAccs {
		workerAccs[w] = newAccs()
	}

	jobs := make(chan *models.StockBrand)
	var processed atomic.Int64
	g, gctx := errgroup.WithContext(ctx)

	for w := 0; w < concurrency; w++ {
		w := w
		g.Go(func() error {
			local := workerAccs[w]
			for brand := range jobs {
				prices, err := r.stockBrandsDailyStockPriceRepository.ListDailyPricesBySymbol(gctx, models.ListDailyPricesBySymbolFilter{
					TickerSymbol: brand.TickerSymbol,
					DateFrom:     &from,
					DateTo:       &to,
					DateOrder:    &asc,
				})
				if err != nil {
					return errors.Wrap(err, "ListDailyPricesBySymbol error for "+brand.TickerSymbol)
				}
				if len(prices) < strategyRankingMinDays {
					continue
				}
				accumulateResults(brand, prices, exitParams, local)
				if n := processed.Add(1); n%200 == 0 {
					log.Printf("strategy ranking: processed %d/%d brands", n, len(brands))
				}
			}
			return nil
		})
	}

	// フィーダ：ctx キャンセルを尊重して銘柄を投入する
	g.Go(func() error {
		defer close(jobs)
		for _, b := range brands {
			select {
			case jobs <- b:
			case <-gctx.Done():
				return gctx.Err()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, int(processed.Load()), err
	}

	// ワーカーローカルの集計をマージ
	accs := newAccs()
	for _, wa := range workerAccs {
		mergeAccs(accs, wa)
	}
	return accs, int(processed.Load()), nil
}

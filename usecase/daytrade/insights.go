package daytrade

import (
	"sort"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// normalizeDirection は旧・新フォーマット両対応で売買方向を返す。
// 旧フォーマット: tradeKind ∈ {売建, 買建, 現物買付, 現物売却}
// 新フォーマット: tradeKind = "" → marginKind を使用
func normalizeDirection(tradeKind, marginKind string) string {
	if tradeKind != "" {
		return tradeKind
	}
	return marginKind
}

type tradeApproxKey struct {
	tickerSymbol string
	executedOn   string // YYYY-MM-DD
	direction    string
}

type tradeAccum struct {
	brandName   string
	executedOn  time.Time
	direction   string
	profitLoss  int64
	tradeAmount int64
	winCount    int
	lossCount   int
}

// BuildTradeApprox は明細を「銘柄×日×売買方向」で集約して1トレード近似を返す。
// 分割決済が複数行に割れている場合に合算する。
// 同一銘柄・同一日に同方向で複数エントリーした場合は区別不能なため合算される（制約として許容）。
func BuildTradeApprox(executions []*models.DaytradeExecution) []*models.DaytradeTradeApprox {
	keys := make([]tradeApproxKey, 0)
	acc := make(map[tradeApproxKey]*tradeAccum)

	for _, ex := range executions {
		dir := normalizeDirection(ex.TradeKind, ex.MarginKind)
		k := tradeApproxKey{
			tickerSymbol: ex.TickerSymbol,
			executedOn:   ex.ExecutedOn.Format("2006-01-02"),
			direction:    dir,
		}
		if _, exists := acc[k]; !exists {
			keys = append(keys, k)
			acc[k] = &tradeAccum{
				brandName:  ex.BrandName,
				executedOn: ex.ExecutedOn,
				direction:  dir,
			}
		}
		a := acc[k]
		a.profitLoss += ex.ProfitLoss
		a.tradeAmount += ex.TradeAmount
		if ex.ProfitLoss > 0 {
			a.winCount++
		} else if ex.ProfitLoss < 0 {
			a.lossCount++
		}
	}

	results := make([]*models.DaytradeTradeApprox, 0, len(keys))
	for _, k := range keys {
		a := acc[k]
		results = append(results, &models.DaytradeTradeApprox{
			TickerSymbol: k.tickerSymbol,
			BrandName:    a.brandName,
			ExecutedOn:   a.executedOn,
			Direction:    k.direction,
			ProfitLoss:   a.profitLoss,
			TradeAmount:  a.tradeAmount,
			WinCount:     a.winCount,
			LossCount:    a.lossCount,
		})
	}
	return results
}

// ComputeInsights はトレード近似スライスから反省指標を計算する。
func ComputeInsights(trades []*models.DaytradeTradeApprox) *models.DaytradeInsights {
	return &models.DaytradeInsights{
		LossConcentration: computeLossConcentration(trades),
		FavoriteTraps:     computeFavoriteTraps(trades),
	}
}

func computeLossConcentration(trades []*models.DaytradeTradeApprox) models.DaytradeLossConcentration {
	lossTrades := make([]*models.DaytradeTradeApprox, 0)
	for _, t := range trades {
		if t.ProfitLoss < 0 {
			lossTrades = append(lossTrades, t)
		}
	}

	// 損失大きい順（profit_loss が最も小さい=最大損失）
	sort.Slice(lossTrades, func(i, j int) bool {
		return lossTrades[i].ProfitLoss < lossTrades[j].ProfitLoss
	})

	var totalLoss int64
	for _, t := range lossTrades {
		totalLoss += -t.ProfitLoss
	}

	worstN := min(5, len(lossTrades))
	worst := make([]models.DaytradeWorstTrade, 0, worstN)
	for i := range worstN {
		t := lossTrades[i]
		worst = append(worst, models.DaytradeWorstTrade{
			TickerSymbol: t.TickerSymbol,
			BrandName:    t.BrandName,
			ExecutedOn:   t.ExecutedOn.Format("2006-01-02"),
			Direction:    t.Direction,
			ProfitLoss:   t.ProfitLoss,
		})
	}

	ratio := func(topN int) float64 {
		if totalLoss == 0 || len(lossTrades) == 0 {
			return 0
		}
		var sum int64
		for i := 0; i < topN && i < len(lossTrades); i++ {
			sum += -lossTrades[i].ProfitLoss
		}
		return float64(sum) / float64(totalLoss)
	}

	return models.DaytradeLossConcentration{
		TotalLoss:   totalLoss,
		Top1Ratio:   ratio(1),
		Top3Ratio:   ratio(3),
		Top5Ratio:   ratio(5),
		WorstTrades: worst,
	}
}

type symbolAcc struct {
	brandName  string
	tradeCount int
	totalPnl   int64
	winCount   int
}

func computeFavoriteTraps(trades []*models.DaytradeTradeApprox) []models.DaytradeFavoriteTrap {
	keyOrder := make([]string, 0)
	acc := make(map[string]*symbolAcc)

	for _, t := range trades {
		if _, exists := acc[t.TickerSymbol]; !exists {
			keyOrder = append(keyOrder, t.TickerSymbol)
			acc[t.TickerSymbol] = &symbolAcc{brandName: t.BrandName}
		}
		a := acc[t.TickerSymbol]
		a.tradeCount++
		a.totalPnl += t.ProfitLoss
		if t.ProfitLoss > 0 {
			a.winCount++
		}
	}

	traps := make([]models.DaytradeFavoriteTrap, 0)
	for _, sym := range keyOrder {
		a := acc[sym]
		if a.totalPnl >= 0 {
			continue // 期待値プラスの銘柄は惚れ込みではない
		}
		expectancy := float64(a.totalPnl) / float64(a.tradeCount)
		winRate := float64(a.winCount) / float64(a.tradeCount)
		traps = append(traps, models.DaytradeFavoriteTrap{
			TickerSymbol: sym,
			BrandName:    a.brandName,
			TradeCount:   a.tradeCount,
			TotalPnl:     a.totalPnl,
			Expectancy:   expectancy,
			WinRate:      winRate,
		})
	}

	// 取引回数の多い順（惚れ込み度順）
	sort.Slice(traps, func(i, j int) bool {
		if traps[i].TradeCount != traps[j].TradeCount {
			return traps[i].TradeCount > traps[j].TradeCount
		}
		return traps[i].TotalPnl < traps[j].TotalPnl // 同数なら損失大きい順
	})

	return traps
}

package daytrade

import (
	"sort"

	"github.com/Code0716/stock-price-repository/models"
)

type tradeNoteKey struct {
	tickerSymbol string
	executedOn   string // YYYY-MM-DD
	direction    string
}

// MergeTradesWithNotes は近似トレードと注釈を近似キーで突き合わせて結合する。
func MergeTradesWithNotes(
	trades []*models.DaytradeTradeApprox,
	notes []*models.DaytradeTradeNoteRecord,
) []*models.DaytradeTradeWithNote {
	noteMap := make(map[tradeNoteKey]*models.DaytradeTradeNoteRecord, len(notes))
	for _, n := range notes {
		k := tradeNoteKey{
			tickerSymbol: n.TickerSymbol,
			executedOn:   n.ExecutedOn.Format("2006-01-02"),
			direction:    n.Direction,
		}
		noteMap[k] = n
	}

	result := make([]*models.DaytradeTradeWithNote, 0, len(trades))
	for _, t := range trades {
		tw := &models.DaytradeTradeWithNote{
			TickerSymbol: t.TickerSymbol,
			BrandName:    t.BrandName,
			ExecutedOn:   t.ExecutedOn.Format("2006-01-02"),
			Direction:    t.Direction,
			ProfitLoss:   t.ProfitLoss,
			TradeAmount:  t.TradeAmount,
		}
		k := tradeNoteKey{
			tickerSymbol: t.TickerSymbol,
			executedOn:   tw.ExecutedOn,
			direction:    t.Direction,
		}
		if n, ok := noteMap[k]; ok {
			tw.Note = &models.DaytradeTradeNote{
				Memo:              n.Memo,
				Tags:              n.Tags,
				DeclaredStopPrice: n.DeclaredStopPrice,
			}
		}
		result = append(result, tw)
	}
	return result
}

type tagAccum struct {
	tradeCount int
	totalPnl   int64
	winCount   int
}

// ComputeTagStats は注釈付きトレードからタグ別損益を集計する。
// タグのないトレードは無視する。TotalPnl 昇順（損失大きい順）で返す。
func ComputeTagStats(trades []*models.DaytradeTradeWithNote) []models.DaytradeTagStat {
	acc := make(map[string]*tagAccum)
	order := make([]string, 0)

	for _, t := range trades {
		if t.Note == nil || len(t.Note.Tags) == 0 {
			continue
		}
		for _, tag := range t.Note.Tags {
			if _, exists := acc[tag]; !exists {
				order = append(order, tag)
				acc[tag] = &tagAccum{}
			}
			a := acc[tag]
			a.tradeCount++
			a.totalPnl += t.ProfitLoss
			if t.ProfitLoss > 0 {
				a.winCount++
			}
		}
	}

	stats := make([]models.DaytradeTagStat, 0, len(order))
	for _, tag := range order {
		a := acc[tag]
		stats = append(stats, models.DaytradeTagStat{
			Tag:        tag,
			TradeCount: a.tradeCount,
			TotalPnl:   a.totalPnl,
			Expectancy: float64(a.totalPnl) / float64(a.tradeCount),
			WinRate:    float64(a.winCount) / float64(a.tradeCount),
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalPnl != stats[j].TotalPnl {
			return stats[i].TotalPnl < stats[j].TotalPnl // 損失大きい順
		}
		return stats[i].TradeCount > stats[j].TradeCount
	})

	return stats
}

package domain_service

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// QuizUniverseValueTopN 売買代金上位で残す銘柄数。
const QuizUniverseValueTopN = 600

// QuizUniverseRangeTopN 値幅率上位で最終的に出題する銘柄数。
const QuizUniverseRangeTopN = 300

type quizUniverseCandidate struct {
	stockBrandID    string
	tickerSymbol    string
	avgTradingValue decimal.Decimal
	avgDailyRange   decimal.Decimal
	baseClosePrice  decimal.Decimal
}

// SelectQuizUniverse 直近営業日群の全銘柄日足から、出来高がありよく動く銘柄を選定する。
// 1) 対象期間の全営業日にバーがある銘柄のみを候補とする（新規上場・売買停止銘柄のノイズ除去）
// 2) 平均売買代金（volume*close）上位 valueTopN に絞る
// 3) その中から平均値幅率（(high-low)/close）上位 rangeTopN を選び、値幅率降順に出題順を割り振る
// quizDate（基準終値日）は入力に含まれる最新の日付とする。
func SelectQuizUniverse(prices []*models.StockBrandDailyPrice, valueTopN, rangeTopN int) []*models.QuizUniverseEntry {
	if len(prices) == 0 {
		return nil
	}

	byBrand := make(map[string][]*models.StockBrandDailyPrice)
	tradingDates := make(map[time.Time]struct{})
	var quizDate time.Time
	for _, p := range prices {
		byBrand[p.StockBrandID] = append(byBrand[p.StockBrandID], p)
		tradingDates[p.Date] = struct{}{}
		if p.Date.After(quizDate) {
			quizDate = p.Date
		}
	}
	requiredDays := len(tradingDates)
	if requiredDays == 0 {
		return nil
	}

	candidates := make([]quizUniverseCandidate, 0, len(byBrand))
	for stockBrandID, rows := range byBrand {
		if len(rows) < requiredDays {
			continue // 期間中に欠損日がある銘柄（新規上場・売買停止等）は除外
		}

		sort.Slice(rows, func(i, j int) bool { return rows[i].Date.Before(rows[j].Date) })

		tradingValueSum := decimal.Zero
		dailyRangeSum := decimal.Zero
		for _, r := range rows {
			if r.Close.IsZero() {
				continue
			}
			tradingValueSum = tradingValueSum.Add(decimal.NewFromInt(r.Volume).Mul(r.Close))
			dailyRangeSum = dailyRangeSum.Add(r.High.Sub(r.Low).Div(r.Close))
		}
		n := decimal.NewFromInt(int64(len(rows)))

		candidates = append(candidates, quizUniverseCandidate{
			stockBrandID:    stockBrandID,
			tickerSymbol:    rows[len(rows)-1].TickerSymbol,
			avgTradingValue: tradingValueSum.Div(n),
			avgDailyRange:   dailyRangeSum.Div(n),
			baseClosePrice:  rows[len(rows)-1].Close,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].avgTradingValue.Equal(candidates[j].avgTradingValue) {
			return candidates[i].avgTradingValue.GreaterThan(candidates[j].avgTradingValue)
		}
		return candidates[i].tickerSymbol < candidates[j].tickerSymbol
	})
	if len(candidates) > valueTopN {
		candidates = candidates[:valueTopN]
	}

	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].avgDailyRange.Equal(candidates[j].avgDailyRange) {
			return candidates[i].avgDailyRange.GreaterThan(candidates[j].avgDailyRange)
		}
		return candidates[i].tickerSymbol < candidates[j].tickerSymbol
	})
	if len(candidates) > rangeTopN {
		candidates = candidates[:rangeTopN]
	}

	entries := make([]*models.QuizUniverseEntry, 0, len(candidates))
	for i, c := range candidates {
		entries = append(entries, &models.QuizUniverseEntry{
			QuizDate:        quizDate,
			StockBrandID:    c.stockBrandID,
			TickerSymbol:    c.tickerSymbol,
			QuestionOrder:   i + 1,
			AvgTradingValue: c.avgTradingValue,
			AvgDailyRange:   c.avgDailyRange,
			BaseClosePrice:  c.baseClosePrice,
		})
	}
	return entries
}

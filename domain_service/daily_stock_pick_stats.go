package domain_service

import (
	"sort"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// DailyPickScoreBandWidth 閲覧用スコア帯の刻み幅。0..100 を10点刻みで固定する。
// 四分位にすると母集団が変わるたび境界が動き、スコアの重み調整の前後比較が成立しないため固定帯にしている。
const DailyPickScoreBandWidth = 10

// dailyPickStatAcc 集計途中の accumulator。平均リターンは horizon ごとに母数（非nil件数）が異なるため個別に持つ。
type dailyPickStatAcc struct {
	total          int
	evaluatedCount int
	win            int
	lose           int
	draw           int
	voidCount      int
	sum1D          decimal.Decimal
	cnt1D          int
	sum3D          decimal.Decimal
	cnt3D          int
	sum5D          decimal.Decimal
	cnt5D          int
}

func (a *dailyPickStatAcc) add(p *models.DailyStockPick) {
	a.total++
	if p.Evaluated() {
		a.evaluatedCount++
	}
	if p.Outcome != nil {
		switch *p.Outcome {
		case models.DailyStockPickOutcomeWin:
			a.win++
		case models.DailyStockPickOutcomeLose:
			a.lose++
		case models.DailyStockPickOutcomeDraw:
			a.draw++
		case models.DailyStockPickOutcomeVoid:
			a.voidCount++
		}
	}
	if p.Return1D != nil {
		a.sum1D = a.sum1D.Add(*p.Return1D)
		a.cnt1D++
	}
	if p.Return3D != nil {
		a.sum3D = a.sum3D.Add(*p.Return3D)
		a.cnt3D++
	}
	if p.Return5D != nil {
		a.sum5D = a.sum5D.Add(*p.Return5D)
		a.cnt5D++
	}
}

func (a *dailyPickStatAcc) summary() models.DailyStockPickStatSummary {
	s := models.DailyStockPickStatSummary{
		Total:          a.total,
		EvaluatedCount: a.evaluatedCount,
		PendingCount:   a.total - a.evaluatedCount,
		Win:            a.win,
		Lose:           a.lose,
		Draw:           a.draw,
		Void:           a.voidCount,
	}
	// 勝率の分母は void を除いた確定件数（SummarizeDailyStockPicks と同一定義）。
	if decided := a.win + a.lose + a.draw; decided > 0 {
		s.WinRate = decimal.NewFromInt(int64(a.win)).Div(decimal.NewFromInt(int64(decided))).Round(4)
	}
	s.AvgReturn1D = avgOrZero(a.sum1D, a.cnt1D)
	s.AvgReturn3D = avgOrZero(a.sum3D, a.cnt3D)
	s.AvgReturn5D = avgOrZero(a.sum5D, a.cnt5D)
	return s
}

// avgOrZero 件数0のときはゼロ除算を避けてゼロを返す。
func avgOrZero(sum decimal.Decimal, count int) decimal.Decimal {
	if count == 0 {
		return decimal.Zero
	}
	return sum.Div(decimal.NewFromInt(int64(count))).Round(6)
}

// SummarizeDailyPicksForView 閲覧用の集計（勝敗内訳・答え合わせ済み件数・勝率・平均1/3/5日リターン）。
// 勝率の分母は win+lose+draw（void 除外）で SummarizeDailyStockPicks と同一定義。
// 平均リターンは各 horizon で非nilのみを分母とする。0件入力ならゼロ値を返す。
func SummarizeDailyPicksForView(picks []*models.DailyStockPick) models.DailyStockPickStatSummary {
	var acc dailyPickStatAcc
	for _, p := range picks {
		acc.add(p)
	}
	return acc.summary()
}

// AggregateDailyPicksByDate 日別サマリを pick_date 昇順で返す。0件入力なら空スライス（nil ではない）。
func AggregateDailyPicksByDate(picks []*models.DailyStockPick) []*models.DailyStockPickDailyStat {
	byDate := make(map[time.Time]*dailyPickStatAcc)
	for _, p := range picks {
		d := dailyPickDateOnly(p.PickDate)
		acc, ok := byDate[d]
		if !ok {
			acc = &dailyPickStatAcc{}
			byDate[d] = acc
		}
		acc.add(p)
	}

	dates := make([]time.Time, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	out := make([]*models.DailyStockPickDailyStat, 0, len(dates))
	for _, d := range dates {
		out = append(out, &models.DailyStockPickDailyStat{
			PickDate:                  d.Format(dailyPickDateLayout),
			DailyStockPickStatSummary: byDate[d].summary(),
		})
	}
	return out
}

// AggregateDailyPicksByScoreBand スコア帯（[0,10),[10,20)...[90,100]）別に集計する。
// score=100 は最終帯に含める。該当0件の帯は返さない（帯は下限昇順）。
// 保存されるのは上位N件のみでスコアが偏るため、空帯まで返すとテーブルがほぼ空行になるので除外している。
func AggregateDailyPicksByScoreBand(picks []*models.DailyStockPick) []*models.DailyStockPickScoreBand {
	byLower := make(map[int]*dailyPickStatAcc)
	for _, p := range picks {
		lower := scoreBandLower(p.Score)
		acc, ok := byLower[lower]
		if !ok {
			acc = &dailyPickStatAcc{}
			byLower[lower] = acc
		}
		acc.add(p)
	}

	lowers := make([]int, 0, len(byLower))
	for l := range byLower {
		lowers = append(lowers, l)
	}
	sort.Ints(lowers)

	out := make([]*models.DailyStockPickScoreBand, 0, len(lowers))
	for _, l := range lowers {
		out = append(out, &models.DailyStockPickScoreBand{
			Band:                      scoreBandLabel(l),
			Lower:                     decimal.NewFromInt(int64(l)),
			Upper:                     decimal.NewFromInt(int64(l + DailyPickScoreBandWidth)),
			DailyStockPickStatSummary: byLower[l].summary(),
		})
	}
	return out
}

// scoreBandLower スコアが属する帯の下限を返す。100 以上は最終帯（90）に含める。負値は 0 帯に丸める。
func scoreBandLower(score decimal.Decimal) int {
	s := score.IntPart()
	if s < 0 {
		return 0
	}
	lower := int(s) / DailyPickScoreBandWidth * DailyPickScoreBandWidth
	if lower >= 100 {
		lower = 100 - DailyPickScoreBandWidth
	}
	return lower
}

// scoreBandLabel "70-79" 形式のラベル。最終帯だけ上限(100)を含むため "90-100" とする。
func scoreBandLabel(lower int) string {
	if lower+DailyPickScoreBandWidth >= 100 {
		return "90-100"
	}
	return strconv.Itoa(lower) + "-" + strconv.Itoa(lower+DailyPickScoreBandWidth-1)
}

const dailyPickDateLayout = "2006-01-02"

// dailyPickDateOnly 時刻部分を落として日付でグルーピングできるようにする。
func dailyPickDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

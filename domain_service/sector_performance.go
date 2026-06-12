package domain_service

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/util"
)

// CalcSectorPerformance 業種別日足データと業種名マップから SectorPerformanceItem スライスを生成する。
// rows は date 昇順・全業種混在で渡す。
// adj_close がゼロの行はスキップ（祝日などで誤って 0 が入っているケースの対処）。
// 結果は periodReturn 降順でソートされる。periodReturn が nil（データ不足）の業種は末尾。
func CalcSectorPerformance(rows []*models.Sector33AverageDailyPrice, names map[string]string) []*models.SectorPerformanceItem {
	// 業種コードごとに昇順スライスを構築
	byCode := make(map[string][]*models.Sector33AverageDailyPrice)
	for _, row := range rows {
		if row.SectorCode == "" {
			continue
		}
		// adj_close がゼロの行をスキップ
		if row.Adjclose.IsZero() {
			continue
		}
		byCode[row.SectorCode] = append(byCode[row.SectorCode], row)
	}

	items := make([]*models.SectorPerformanceItem, 0, len(byCode))
	for code, series := range byCode {
		name, ok := names[code]
		if !ok {
			// マップにない場合はコードをそのまま使用
			name = code
		}

		item := calcItem(code, name, series)
		items = append(items, item)
	}

	// periodReturn 降順ソート。nil（データ不足）は末尾。
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i].PeriodReturn, items[j].PeriodReturn
		if a == nil && b == nil {
			return items[i].SectorCode < items[j].SectorCode
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.GreaterThan(*b)
	})

	return items
}

func calcItem(code, name string, series []*models.Sector33AverageDailyPrice) *models.SectorPerformanceItem {
	item := &models.SectorPerformanceItem{
		SectorCode: code,
		SectorName: name,
	}

	if len(series) == 0 {
		return item
	}

	latest := series[len(series)-1]
	latestClose := latest.Adjclose.Round(4)
	item.LatestClose = &latestClose
	item.LatestDate = latest.Date.Format(util.DateLayout)

	// periodReturn = 期間内最初の adjClose 比
	first := series[0]
	if !first.Adjclose.IsZero() {
		pr := latest.Adjclose.Div(first.Adjclose).Sub(decimal.NewFromInt(1)).Round(6)
		item.PeriodReturn = &pr
	}

	// return5d = 直近 5 本前（index: len-1-5=len-6）の adjClose 比
	item.Return5d = calcNDayReturn(series, 5)

	// return25d = 直近 25 本前の adjClose 比
	item.Return25d = calcNDayReturn(series, 25)

	return item
}

// calcNDayReturn 系列末尾から n 本前の adjClose に対するリターンを計算する。
// 系列の長さが n+1 本未満（n 本前が存在しない）の場合は nil を返す。
func calcNDayReturn(series []*models.Sector33AverageDailyPrice, n int) *decimal.Decimal {
	length := len(series)
	if length <= n {
		return nil
	}
	base := series[length-1-n].Adjclose
	if base.IsZero() {
		return nil
	}
	latest := series[length-1].Adjclose
	r := latest.Div(base).Sub(decimal.NewFromInt(1)).Round(6)
	return &r
}

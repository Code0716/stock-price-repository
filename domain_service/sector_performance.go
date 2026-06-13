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
	byCode := make(map[string][]*models.Sector33AverageDailyPrice)
	for _, row := range rows {
		if row.SectorCode == "" {
			continue
		}
		if row.Adjclose.IsZero() {
			continue
		}
		byCode[row.SectorCode] = append(byCode[row.SectorCode], row)
	}

	items := make([]*models.SectorPerformanceItem, 0, len(byCode))
	for code, series := range byCode {
		name, ok := names[code]
		if !ok {
			name = code
		}
		items = append(items, calcItem(code, name, series))
	}

	return sortItems(items)
}

// CalcSector17Performance 17業種平均日足データと業種名マップから SectorPerformanceItem スライスを生成する。
// rows は date 昇順・全業種混在で渡す。
// adj_close がゼロの行はスキップ。
// 結果は periodReturn 降順でソートされる。periodReturn が nil（データ不足）の業種は末尾。
func CalcSector17Performance(rows []*models.Sector17AverageDailyPrice, names map[string]string) []*models.SectorPerformanceItem {
	byCode := make(map[string][]*models.Sector17AverageDailyPrice)
	for _, row := range rows {
		if row.SectorCode == "" {
			continue
		}
		if row.Adjclose.IsZero() {
			continue
		}
		byCode[row.SectorCode] = append(byCode[row.SectorCode], row)
	}

	items := make([]*models.SectorPerformanceItem, 0, len(byCode))
	for code, series := range byCode {
		name, ok := names[code]
		if !ok {
			name = code
		}
		items = append(items, calcItem17(code, name, series))
	}

	return sortItems(items)
}

func sortItems(items []*models.SectorPerformanceItem) []*models.SectorPerformanceItem {
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

	first := series[0]
	if !first.Adjclose.IsZero() {
		pr := latest.Adjclose.Div(first.Adjclose).Sub(decimal.NewFromInt(1)).Round(6)
		item.PeriodReturn = &pr
	}

	item.Return5d = calcNDayReturn33(series, 5)
	item.Return25d = calcNDayReturn33(series, 25)

	return item
}

func calcItem17(code, name string, series []*models.Sector17AverageDailyPrice) *models.SectorPerformanceItem {
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

	first := series[0]
	if !first.Adjclose.IsZero() {
		pr := latest.Adjclose.Div(first.Adjclose).Sub(decimal.NewFromInt(1)).Round(6)
		item.PeriodReturn = &pr
	}

	item.Return5d = calcNDayReturn17(series, 5)
	item.Return25d = calcNDayReturn17(series, 25)

	return item
}

func calcNDayReturn33(series []*models.Sector33AverageDailyPrice, n int) *decimal.Decimal {
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

func calcNDayReturn17(series []*models.Sector17AverageDailyPrice, n int) *decimal.Decimal {
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

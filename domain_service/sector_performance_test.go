package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func mustParseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func makeRow(date string, code string, adjclose float64) *models.Sector33AverageDailyPrice {
	return &models.Sector33AverageDailyPrice{
		Date:       mustParseDate(date),
		SectorCode: code,
		Open:       decimal.NewFromFloat(adjclose),
		Close:      decimal.NewFromFloat(adjclose),
		High:       decimal.NewFromFloat(adjclose),
		Low:        decimal.NewFromFloat(adjclose),
		Adjclose:   decimal.NewFromFloat(adjclose),
	}
}

func TestCalcSectorPerformance_PeriodReturn(t *testing.T) {
	names := map[string]string{
		"3700": "輸送用機器",
	}

	rows := []*models.Sector33AverageDailyPrice{
		makeRow("2024-01-04", "3700", 1000.0),
		makeRow("2024-01-05", "3700", 1050.0),
		makeRow("2024-01-08", "3700", 1100.0),
	}

	items := CalcSectorPerformance(rows, names)
	assert.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, "3700", item.SectorCode)
	assert.Equal(t, "輸送用機器", item.SectorName)
	assert.NotNil(t, item.PeriodReturn)

	// periodReturn = 1100 / 1000 - 1 = 0.1
	expected := decimal.NewFromFloat(0.1)
	assert.True(t, expected.Equal(item.PeriodReturn.Round(6)), "periodReturn: %s vs %s", expected, item.PeriodReturn)

	// 3本 → 5d 不足 → nil
	assert.Nil(t, item.Return5d, "return5d should be nil when insufficient data")
}

func TestCalcSectorPerformance_Return5d(t *testing.T) {
	names := map[string]string{
		"3700": "輸送用機器",
	}

	// 7本用意（5d以上、25d未満）
	rows := make([]*models.Sector33AverageDailyPrice, 7)
	for i := range 7 {
		rows[i] = makeRow(
			time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			"3700",
			float64(1000+i*10),
		)
	}

	items := CalcSectorPerformance(rows, names)
	assert.Len(t, items, 1)
	item := items[0]

	assert.NotNil(t, item.Return5d, "return5d should not be nil")
	assert.Nil(t, item.Return25d, "return25d should be nil when insufficient data")

	// rows[6]=1060, rows[1]=1010: (1060/1010)-1
	expected := decimal.NewFromFloat(1060).Div(decimal.NewFromFloat(1010)).Sub(decimal.NewFromInt(1)).Round(6)
	assert.True(t, expected.Equal(*item.Return5d), "return5d: got %s, want %s", item.Return5d, expected)
}

func TestCalcSectorPerformance_ZeroAdjcloseSkip(t *testing.T) {
	names := map[string]string{
		"3700": "輸送用機器",
	}

	// adj_close がゼロの行はスキップされ、2行だけ有効 → periodReturn あり、5d/25d は nil
	rows := []*models.Sector33AverageDailyPrice{
		makeRow("2024-01-04", "3700", 0.0),  // ゼロ → スキップ
		makeRow("2024-01-05", "3700", 1000.0),
		makeRow("2024-01-08", "3700", 1100.0),
	}

	items := CalcSectorPerformance(rows, names)
	assert.Len(t, items, 1)
	item := items[0]

	assert.NotNil(t, item.PeriodReturn)
	// first有効行は 1000 → period = 1100/1000 - 1 = 0.1
	expected := decimal.NewFromFloat(0.1)
	assert.True(t, expected.Equal(item.PeriodReturn.Round(6)))
}

func TestCalcSectorPerformance_Sort(t *testing.T) {
	names := map[string]string{
		"3700": "輸送用機器",
		"3650": "電気機器",
		"3600": "機械",
	}

	// 3700: +10%, 3650: +5%, 3600: +20%
	rows := []*models.Sector33AverageDailyPrice{
		makeRow("2024-01-04", "3700", 1000.0),
		makeRow("2024-01-05", "3700", 1100.0),
		makeRow("2024-01-04", "3650", 1000.0),
		makeRow("2024-01-05", "3650", 1050.0),
		makeRow("2024-01-04", "3600", 1000.0),
		makeRow("2024-01-05", "3600", 1200.0),
	}

	items := CalcSectorPerformance(rows, names)
	assert.Len(t, items, 3)

	// periodReturn降順 → 3600 (+20%), 3700 (+10%), 3650 (+5%)
	assert.Equal(t, "3600", items[0].SectorCode)
	assert.Equal(t, "3700", items[1].SectorCode)
	assert.Equal(t, "3650", items[2].SectorCode)
}

func TestCalcSectorPerformance_EmptyRows(t *testing.T) {
	names := map[string]string{"3700": "輸送用機器"}
	items := CalcSectorPerformance(nil, names)
	assert.Empty(t, items)
}

func TestCalcSectorPerformance_NoDataForSector(t *testing.T) {
	// 全行がゼロ adjclose → 有効データなし → sectors 空
	names := map[string]string{"3700": "輸送用機器"}
	rows := []*models.Sector33AverageDailyPrice{
		makeRow("2024-01-04", "3700", 0.0),
		makeRow("2024-01-05", "3700", 0.0),
	}
	items := CalcSectorPerformance(rows, names)
	assert.Empty(t, items)
}

func TestCalcNDayReturn_InsufficientData(t *testing.T) {
	series := []*models.Sector33AverageDailyPrice{
		makeRow("2024-01-04", "3700", 1000.0),
		makeRow("2024-01-05", "3700", 1050.0),
	}
	// 2本 → n=5 は不足
	result := calcNDayReturn(series, 5)
	assert.Nil(t, result)
}

func TestCalcNDayReturn_Exact(t *testing.T) {
	// 6本 → n=5 はギリギリ存在（series[0] vs series[5]）
	series := make([]*models.Sector33AverageDailyPrice, 6)
	for i := range 6 {
		series[i] = makeRow(
			time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			"3700",
			float64(1000+i*10),
		)
	}
	result := calcNDayReturn(series, 5)
	assert.NotNil(t, result)
	// series[5]=1050, series[0]=1000: (1050/1000)-1 = 0.05
	expected := decimal.NewFromFloat(0.05)
	assert.True(t, expected.Equal(result.Round(6)), "got %s, want %s", result, expected)
}

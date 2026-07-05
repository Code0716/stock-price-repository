package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

// buildDailyPrices baseDate から days 日分の連続する日足データを生成する（Close は 100+i）。
func buildDailyPrices(baseDate time.Time, days int) []*models.StockBrandDailyPrice {
	prices := make([]*models.StockBrandDailyPrice, days)
	for i := 0; i < days; i++ {
		prices[i] = &models.StockBrandDailyPrice{
			Date:   baseDate.AddDate(0, 0, i),
			Open:   decimal.NewFromInt(int64(100 + i)),
			High:   decimal.NewFromInt(int64(110 + i)),
			Low:    decimal.NewFromInt(int64(90 + i)),
			Close:  decimal.NewFromInt(int64(100 + i)),
			Volume: 1000,
		}
	}
	return prices
}

func TestBuildDailyChartSeries(t *testing.T) {
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	type args struct {
		prices      []*models.StockBrandDailyPrice
		visibleFrom time.Time
	}
	tests := []struct {
		name          string
		args          args
		wantCandles   int
		wantMA5Count  int
		wantMA25Count int
		wantMA75Count int
	}{
		{
			name: "正常系: visibleFromより前のデータは除外される",
			args: args{
				// 100日分のうち、後半10日だけを可視範囲とする
				prices:      buildDailyPrices(baseDate, 100),
				visibleFrom: baseDate.AddDate(0, 0, 90),
			},
			wantCandles:   10,
			wantMA5Count:  10, // 可視範囲は十分に後方のためMA5もすべて非ゼロ
			wantMA25Count: 10,
			wantMA75Count: 10,
		},
		{
			name: "正常系: visibleFromがゼロ値の場合は全点を含む",
			args: args{
				prices:      buildDailyPrices(baseDate, 5),
				visibleFrom: time.Time{},
			},
			wantCandles:   5,
			wantMA5Count:  1, // 5日目のみMA5が計算可能
			wantMA25Count: 0,
			wantMA75Count: 0,
		},
		{
			name: "正常系: pricesが空の場合は空のチャートを返す",
			args: args{
				prices:      []*models.StockBrandDailyPrice{},
				visibleFrom: time.Time{},
			},
			wantCandles:   0,
			wantMA5Count:  0,
			wantMA25Count: 0,
			wantMA75Count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDailyChartSeries(tt.args.prices, tt.args.visibleFrom)

			assert.Len(t, got.Candles, tt.wantCandles)
			assert.Len(t, got.MA5, tt.wantMA5Count)
			assert.Len(t, got.MA25, tt.wantMA25Count)
			assert.Len(t, got.MA75, tt.wantMA75Count)

			if tt.wantCandles > 0 {
				firstVisible := tt.args.prices[len(tt.args.prices)-tt.wantCandles]
				assert.Equal(t, firstVisible.Date.Format("2006-01-02"), got.Candles[0].Date)
				assert.True(t, got.Candles[0].Close.Equal(firstVisible.Close))
			}
		})
	}
}

package domain_service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func TestBuildQuizChartSeries(t *testing.T) {
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	type args struct {
		prices      []*models.StockBrandDailyPrice
		visibleFrom time.Time
	}
	tests := []struct {
		name         string
		args         args
		wantQuizDate string
		wantCandles  int
	}{
		{
			name: "正常系: QuizDateはpricesの最終日になる",
			args: args{
				prices:      buildDailyPrices(baseDate, 10),
				visibleFrom: baseDate,
			},
			wantQuizDate: baseDate.AddDate(0, 0, 9).Format("2006-01-02"),
			wantCandles:  10,
		},
		{
			name: "正常系: pricesが空の場合はQuizDateが空文字になる",
			args: args{
				prices:      []*models.StockBrandDailyPrice{},
				visibleFrom: time.Time{},
			},
			wantQuizDate: "",
			wantCandles:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildQuizChartSeries(tt.args.prices, tt.args.visibleFrom)

			assert.Equal(t, tt.wantQuizDate, got.QuizDate)
			assert.Len(t, got.Candles, tt.wantCandles)
		})
	}
}

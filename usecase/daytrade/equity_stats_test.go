package daytrade

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

func buckets(pls ...int64) []*models.DaytradeSummaryBucket {
	result := make([]*models.DaytradeSummaryBucket, 0, len(pls))
	for _, pl := range pls {
		result = append(result, &models.DaytradeSummaryBucket{ProfitLoss: pl})
	}
	return result
}

func TestComputeEquityStats(t *testing.T) {
	tests := []struct {
		name            string
		daily           []*models.DaytradeSummaryBucket
		wantMaxDrawdown int64
		wantMaxStreak   int
	}{
		{
			name:            "空スライス",
			daily:           buckets(),
			wantMaxDrawdown: 0,
			wantMaxStreak:   0,
		},
		{
			name:            "全勝（ドローダウンなし・連敗なし）",
			daily:           buckets(100, 200, 300),
			wantMaxDrawdown: 0,
			wantMaxStreak:   0,
		},
		{
			name:            "全敗（ピーク0から即ドローダウン）",
			daily:           buckets(-100, -200, -300),
			wantMaxDrawdown: 600, // cumulative: -100, -300, -600 / peak=0 → max DD=600
			wantMaxStreak:   3,
		},
		{
			name:            "最初が利益でその後下落",
			daily:           buckets(1000, -400, -200),
			wantMaxDrawdown: 600, // peak=1000, cumulative=400→最小 / DD=600
			wantMaxStreak:   2,
		},
		{
			name:            "連敗が複数回ある場合は最大を返す",
			daily:           buckets(-100, 50, -200, -300, 100, -50),
			wantMaxStreak:   2,   // -200,-300 の 2 連敗が最大
			wantMaxDrawdown: 550, // peak=0, 最低 cumulative=-550 → DD=550
		},
		{
			name:            "ドローダウンが複数の谷を持つ場合",
			daily:           buckets(1000, -800, 500, -1200),
			wantMaxDrawdown: 1500, // peak=1000 → cum=-500(day4) → DD=1500
			wantMaxStreak:   1,
		},
		{
			name:            "ブレイクイーブン（profitLoss==0）は連敗をリセット",
			daily:           buckets(-100, 0, -200),
			wantMaxDrawdown: 300, // cumulative: -100, -100, -300 / peak=0 → DD=300
			wantMaxStreak:   1,   // 0 が連敗をリセットする
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDD, gotStreak := ComputeEquityStats(tt.daily)
			assert.Equal(t, tt.wantMaxDrawdown, gotDD, "maxDrawdown")
			assert.Equal(t, tt.wantMaxStreak, gotStreak, "maxLossStreak")
		})
	}
}

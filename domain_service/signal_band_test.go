package domain_service

import (
	"testing"

	"github.com/Code0716/stock-price-repository/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- テストヘルパー ----

func intPtr(v int) *int { return &v }

func decPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

func signalWithRank(rank *int) *models.EvaluatedSignal {
	r5 := decimal.NewFromFloat(0.05)
	return &models.EvaluatedSignal{
		Method:     "test",
		SignalRank: rank,
		Returns:    map[int]*decimal.Decimal{5: &r5},
	}
}

func signalWithScore(score *decimal.Decimal, ret float64) *models.EvaluatedSignal {
	r5 := decimal.NewFromFloat(ret)
	return &models.EvaluatedSignal{
		Method:  "test",
		Score:   score,
		Returns: map[int]*decimal.Decimal{5: &r5},
	}
}

// ---- AggregateByRankBand テスト ----

func TestAggregateByRankBand_BandNames(t *testing.T) {
	// 帯名と順序の確認（シグナル0件でも3帯返る）
	result := AggregateByRankBand([]*models.EvaluatedSignal{}, []int{5})
	require.Len(t, result, 3)
	assert.Equal(t, "1-3", result[0].Band)
	assert.Equal(t, "4-10", result[1].Band)
	assert.Equal(t, "11+", result[2].Band)
}

func TestAggregateByRankBand_EmptyBands(t *testing.T) {
	// シグナル0件でも SignalCount=0 で3帯返る
	result := AggregateByRankBand([]*models.EvaluatedSignal{}, []int{5})
	for _, b := range result {
		assert.Equal(t, 0, b.SignalCount)
	}
}

func TestAggregateByRankBand_BoundaryRank3to4(t *testing.T) {
	// 境界: rank=3 は "1-3"、rank=4 は "4-10"
	signals := []*models.EvaluatedSignal{
		signalWithRank(intPtr(3)),
		signalWithRank(intPtr(4)),
	}
	result := AggregateByRankBand(signals, []int{5})
	require.Len(t, result, 3)
	assert.Equal(t, 1, result[0].SignalCount, "rank=3 は 1-3 帯")
	assert.Equal(t, 1, result[1].SignalCount, "rank=4 は 4-10 帯")
	assert.Equal(t, 0, result[2].SignalCount, "11+ 帯は空")
}

func TestAggregateByRankBand_BoundaryRank10to11(t *testing.T) {
	// 境界: rank=10 は "4-10"、rank=11 は "11+"
	signals := []*models.EvaluatedSignal{
		signalWithRank(intPtr(10)),
		signalWithRank(intPtr(11)),
	}
	result := AggregateByRankBand(signals, []int{5})
	require.Len(t, result, 3)
	assert.Equal(t, 0, result[0].SignalCount, "1-3 帯は空")
	assert.Equal(t, 1, result[1].SignalCount, "rank=10 は 4-10 帯")
	assert.Equal(t, 1, result[2].SignalCount, "rank=11 は 11+ 帯")
}

func TestAggregateByRankBand_NilRankExcluded(t *testing.T) {
	// SignalRank=nil のシグナルは除外
	signals := []*models.EvaluatedSignal{
		signalWithRank(intPtr(1)),
		signalWithRank(nil), // 除外
	}
	result := AggregateByRankBand(signals, []int{5})
	total := 0
	for _, b := range result {
		total += b.SignalCount
	}
	assert.Equal(t, 1, total, "nil rank は除外されること")
}

func TestAggregateByRankBand_ReturnsNilSkipped(t *testing.T) {
	// Returns=nil（skip）のシグナルは SignalCount に含まれるが stats の evaluatedCount には入らない
	signals := []*models.EvaluatedSignal{
		{Method: "test", SignalRank: intPtr(1), Returns: nil}, // skip
		signalWithRank(intPtr(2)),
	}
	result := AggregateByRankBand(signals, []int{5})
	assert.Equal(t, 2, result[0].SignalCount, "skip を含む SignalCount")
	assert.Equal(t, 1, result[0].Stats[5].EvaluatedCount, "skip は evaluatedCount に含まれない")
}

// ---- AggregateByScoreQuartile テスト ----

func TestAggregateByScoreQuartile_NLessThan4_ReturnsEmpty(t *testing.T) {
	// n<4 のとき空を返す
	signals := []*models.EvaluatedSignal{
		signalWithScore(decPtr(0.5), 0.1),
		signalWithScore(decPtr(0.6), 0.2),
		signalWithScore(decPtr(0.7), 0.3),
	}
	result := AggregateByScoreQuartile(signals, []int{5})
	assert.Empty(t, result)
}

func TestAggregateByScoreQuartile_NilScoreExcluded(t *testing.T) {
	// Score=nil のシグナルは除外（全部 nil なら空）
	signals := []*models.EvaluatedSignal{
		signalWithScore(nil, 0.1),
		signalWithScore(nil, 0.2),
		signalWithScore(nil, 0.3),
		signalWithScore(nil, 0.4),
	}
	result := AggregateByScoreQuartile(signals, []int{5})
	assert.Empty(t, result)
}

func TestAggregateByScoreQuartile_N8_2PerQuartile(t *testing.T) {
	// n=8 で各四分位 2件ずつ
	signals := make([]*models.EvaluatedSignal, 8)
	for i := 0; i < 8; i++ {
		signals[i] = signalWithScore(decPtr(float64(i+1)*0.1), 0.01*float64(i+1))
	}
	result := AggregateByScoreQuartile(signals, []int{5})
	require.Len(t, result, 4)
	for i, b := range result {
		assert.Equal(t, "Q"+string(rune('1'+i)), b.Band)
		assert.Equal(t, 2, b.SignalCount, "各四分位は2件")
	}
}

func TestAggregateByScoreQuartile_Q1LowQ4High(t *testing.T) {
	// Q1=低スコア、Q4=高スコア の順序確認
	// score: 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8
	// Q1=[0.1,0.2], Q4=[0.7,0.8]
	signals := []*models.EvaluatedSignal{
		signalWithScore(decPtr(0.8), 0.08), // 高スコア
		signalWithScore(decPtr(0.1), 0.01), // 低スコア
		signalWithScore(decPtr(0.5), 0.05),
		signalWithScore(decPtr(0.2), 0.02),
		signalWithScore(decPtr(0.6), 0.06),
		signalWithScore(decPtr(0.3), 0.03),
		signalWithScore(decPtr(0.7), 0.07),
		signalWithScore(decPtr(0.4), 0.04),
	}
	result := AggregateByScoreQuartile(signals, []int{5})
	require.Len(t, result, 4)
	assert.Equal(t, "Q1", result[0].Band)
	assert.Equal(t, "Q4", result[3].Band)

	// Q1 の 2シグナルは score=0.1, 0.2 → returns = 0.01, 0.02 → avg 低い
	// Q4 の 2シグナルは score=0.7, 0.8 → returns = 0.07, 0.08 → avg 高い
	q1Avg := result[0].Stats[5].AvgReturn
	q4Avg := result[3].Stats[5].AvgReturn
	assert.True(t, q1Avg.LessThan(q4Avg), "Q1 avg < Q4 avg (Q4は高スコア)")
}

func TestAggregateByScoreQuartile_BandNames(t *testing.T) {
	// n=4 で各四分位 1件ずつ、Band名確認
	signals := []*models.EvaluatedSignal{
		signalWithScore(decPtr(0.1), 0.1),
		signalWithScore(decPtr(0.2), 0.2),
		signalWithScore(decPtr(0.3), 0.3),
		signalWithScore(decPtr(0.4), 0.4),
	}
	result := AggregateByScoreQuartile(signals, []int{5})
	require.Len(t, result, 4)
	assert.Equal(t, "Q1", result[0].Band)
	assert.Equal(t, "Q2", result[1].Band)
	assert.Equal(t, "Q3", result[2].Band)
	assert.Equal(t, "Q4", result[3].Band)
}

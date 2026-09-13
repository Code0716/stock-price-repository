package domain_service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/models"
)

// statPick 集計テスト用の推奨1件を組み立てる。returns は nil を渡すと未確定扱い。
func statPick(pickDate time.Time, score string, outcome *models.DailyStockPickOutcome, r1d, r3d, r5d *string) *models.DailyStockPick {
	p := &models.DailyStockPick{
		PickDate: pickDate,
		Score:    decimal.RequireFromString(score),
		Outcome:  outcome,
	}
	if outcome != nil {
		evaluatedAt := pickDate
		p.EvaluatedAt = &evaluatedAt
	}
	if r1d != nil {
		v := decimal.RequireFromString(*r1d)
		p.Return1D = &v
	}
	if r3d != nil {
		v := decimal.RequireFromString(*r3d)
		p.Return3D = &v
	}
	if r5d != nil {
		v := decimal.RequireFromString(*r5d)
		p.Return5D = &v
	}
	return p
}

func statOutcome(o models.DailyStockPickOutcome) *models.DailyStockPickOutcome { return &o }

func statStr(s string) *string { return &s }

func TestScoreBandLower(t *testing.T) {
	tests := []struct {
		name  string
		score string
		want  int
	}{
		{name: "0は0帯", score: "0", want: 0},
		{name: "9.99は0帯", score: "9.99", want: 0},
		{name: "10は10帯", score: "10", want: 10},
		{name: "帯の下限ちょうど(60)", score: "60", want: 60},
		{name: "帯の上限側(69.99)", score: "69.99", want: 60},
		{name: "70は70帯", score: "70", want: 70},
		{name: "100は最終帯(90)に含める", score: "100", want: 90},
		{name: "100超も最終帯に丸める", score: "120", want: 90},
		{name: "負値は0帯に丸める", score: "-5", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scoreBandLower(decimal.RequireFromString(tt.score)))
		})
	}
}

func TestScoreBandLabel(t *testing.T) {
	assert.Equal(t, "0-9", scoreBandLabel(0))
	assert.Equal(t, "60-69", scoreBandLabel(60))
	assert.Equal(t, "90-100", scoreBandLabel(90), "最終帯だけ上限100を含む")
}

func TestSummarizeDailyPicksForView(t *testing.T) {
	d := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("0件入力はゼロ値", func(t *testing.T) {
		s := SummarizeDailyPicksForView(nil)
		assert.Equal(t, 0, s.Total)
		assert.Equal(t, 0, s.PendingCount)
		assert.True(t, s.WinRate.IsZero())
		assert.True(t, s.AvgReturn5D.IsZero())
	})

	t.Run("勝率の分母からvoidを除外する", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.05")),
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.03")),
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeLose), nil, nil, statStr("-0.02")),
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeDraw), nil, nil, statStr("0")),
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeVoid), nil, nil, nil),
		}
		s := SummarizeDailyPicksForView(picks)
		assert.Equal(t, 5, s.Total)
		assert.Equal(t, 2, s.Win)
		assert.Equal(t, 1, s.Lose)
		assert.Equal(t, 1, s.Draw)
		assert.Equal(t, 1, s.Void)
		// 分母は win+lose+draw=4 なので 2/4=0.5（voidを含めると 2/5=0.4 になってしまう）
		assert.Equal(t, "0.5", s.WinRate.String())
	})

	t.Run("平均リターンはhorizonごとに非nilのみを分母にする", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeWin), statStr("0.02"), nil, statStr("0.10")),
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeWin), statStr("0.04"), nil, nil),
		}
		s := SummarizeDailyPicksForView(picks)
		// 1D は2件平均 = 0.03
		assert.Equal(t, "0.03", s.AvgReturn1D.String())
		// 3D は0件なのでゼロ（nilをゼロとして数えない）
		assert.True(t, s.AvgReturn3D.IsZero())
		// 5D は1件平均 = 0.1
		assert.Equal(t, "0.1", s.AvgReturn5D.String())
	})

	t.Run("答え合わせ済み件数と判定待ち件数", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			statPick(d, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.05")),
			statPick(d, "80", nil, nil, nil, nil),
			statPick(d, "80", nil, nil, nil, nil),
		}
		s := SummarizeDailyPicksForView(picks)
		assert.Equal(t, 3, s.Total)
		assert.Equal(t, 1, s.EvaluatedCount)
		assert.Equal(t, 2, s.PendingCount)
	})
}

func TestAggregateDailyPicksByDate(t *testing.T) {
	d1 := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("0件入力は空スライス（nilではない）", func(t *testing.T) {
		got := AggregateDailyPicksByDate(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("pick_date昇順で返す", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			statPick(d3, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.05")),
			statPick(d1, "80", statOutcome(models.DailyStockPickOutcomeLose), nil, nil, statStr("-0.01")),
			statPick(d2, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.02")),
			statPick(d1, "80", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.03")),
		}
		got := AggregateDailyPicksByDate(picks)
		assert.Len(t, got, 3)
		assert.Equal(t, "2026-07-22", got[0].PickDate)
		assert.Equal(t, "2026-07-23", got[1].PickDate)
		assert.Equal(t, "2026-07-24", got[2].PickDate)
		// 7-22 は 1勝1敗
		assert.Equal(t, 2, got[0].Total)
		assert.Equal(t, 1, got[0].Win)
		assert.Equal(t, 1, got[0].Lose)
	})

	t.Run("時刻部分が異なっても同じ日にまとまる", func(t *testing.T) {
		withTime := time.Date(2026, 7, 22, 15, 30, 0, 0, time.UTC)
		picks := []*models.DailyStockPick{
			statPick(d1, "80", nil, nil, nil, nil),
			statPick(withTime, "80", nil, nil, nil, nil),
		}
		got := AggregateDailyPicksByDate(picks)
		assert.Len(t, got, 1)
		assert.Equal(t, 2, got[0].Total)
	})
}

func TestAggregateDailyPicksByScoreBand(t *testing.T) {
	d := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	t.Run("0件入力は空スライス（nilではない）", func(t *testing.T) {
		got := AggregateDailyPicksByScoreBand(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("帯は下限昇順・該当0件の帯は返さない", func(t *testing.T) {
		picks := []*models.DailyStockPick{
			statPick(d, "85", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.05")),
			statPick(d, "62", statOutcome(models.DailyStockPickOutcomeLose), nil, nil, statStr("-0.02")),
			statPick(d, "88", statOutcome(models.DailyStockPickOutcomeWin), nil, nil, statStr("0.07")),
		}
		got := AggregateDailyPicksByScoreBand(picks)
		// 60帯と80帯のみ。70帯など該当0件の帯は返さない
		assert.Len(t, got, 2)
		assert.Equal(t, "60-69", got[0].Band)
		assert.Equal(t, "80-89", got[1].Band)
		assert.Equal(t, 1, got[0].Total)
		assert.Equal(t, 2, got[1].Total)
		// 80帯は2勝0敗
		assert.Equal(t, "1", got[1].WinRate.String())
	})

	t.Run("Lower/Upperが帯の範囲を表す", func(t *testing.T) {
		picks := []*models.DailyStockPick{statPick(d, "75", nil, nil, nil, nil)}
		got := AggregateDailyPicksByScoreBand(picks)
		assert.Len(t, got, 1)
		assert.Equal(t, "70", got[0].Lower.String())
		assert.Equal(t, "80", got[0].Upper.String())
	})

	t.Run("score=100は最終帯に入る", func(t *testing.T) {
		picks := []*models.DailyStockPick{statPick(d, "100", nil, nil, nil, nil)}
		got := AggregateDailyPicksByScoreBand(picks)
		assert.Len(t, got, 1)
		assert.Equal(t, "90-100", got[0].Band)
	})
}

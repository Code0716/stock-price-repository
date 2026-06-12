package usecase

import (
	"context"
	"testing"
	"time"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// --- テストヘルパー ---

func spDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func spSignal(symbol, method, action string, date time.Time) *models.AnalyzeStockBrandPriceHistory {
	return &models.AnalyzeStockBrandPriceHistory{
		ID:           "id-" + symbol,
		TickerSymbol: symbol,
		Name:         symbol + "株式会社",
		Method:       method,
		Action:       action,
		CreatedAt:    date,
	}
}

func spPrice(symbol string, date time.Time, adjclose float64) *models.StockBrandDailyPrice {
	return &models.StockBrandDailyPrice{
		TickerSymbol: symbol,
		Date:         date,
		Adjclose:     decimal.NewFromFloat(adjclose),
	}
}

// --- GetSignalPerformance テスト ---

func TestSignalPerformanceInteractor_GetSignalPerformance(t *testing.T) {
	from := spDate(2024, 1, 4)
	to := spDate(2024, 3, 31)
	// to+40日
	wantPriceTo := to.AddDate(0, 0, signalPerformancePriceLookAhead)

	signalDate := spDate(2024, 1, 10)
	method1 := models.AnalyzeStockBrandPriceHistoryMethodFindMACDBullishV1
	method2 := models.AnalyzeStockBrandPriceHistoryMethodFindTriangleV1

	// シグナル日から始まる価格（昇順、25本以上用意して horizon=20 まで評価可能に）
	makePrices := func(symbol string, start time.Time, n int, base float64) []*models.StockBrandDailyPrice {
		prices := make([]*models.StockBrandDailyPrice, n)
		for i := 0; i < n; i++ {
			prices[i] = spPrice(symbol, start.AddDate(0, 0, i), base+float64(i))
		}
		return prices
	}

	type fields struct {
		analyzeRepo func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository
		priceRepo   func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository
	}

	tests := []struct {
		name    string
		filter  *models.SignalPerformanceFilter
		fields  fields
		wantErr bool
		check   func(t *testing.T, got *models.SignalPerformance)
	}{
		{
			name: "正常系: 複数手法のシグナルを集計し summaries が返る / priceTo が to+40日であること（P1回帰）",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Eq(&models.SignalPerformanceFilter{
						From: from,
						To:   to,
					})).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignal("7203", method1, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate),
						spSignal("6758", method2, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, f models.ListRangePricesBySymbolsFilter) ([]*models.StockBrandDailyPrice, error) {
							// P1回帰: DateTo が to+40日であること
							assert.NotNil(t, f.DateTo)
							assert.Equal(t, wantPriceTo.Truncate(24*time.Hour), f.DateTo.Truncate(24*time.Hour))
							assert.Len(t, f.Symbols, 2)

							// 各銘柄 30 本分の価格を返す
							var prices []*models.StockBrandDailyPrice
							prices = append(prices, makePrices("7203", signalDate, 30, 1000)...)
							prices = append(prices, makePrices("6758", signalDate, 30, 2000)...)
							return prices, nil
						},
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				assert.Equal(t, from, got.From)
				assert.Equal(t, to, got.To)
				assert.ElementsMatch(t, signalPerformanceHorizons, got.Horizons)
				assert.Len(t, got.Summaries, 2)
				// method 未指定なので Signals は空
				assert.Empty(t, got.Signals)
				// 各サマリは SignalCount=1 / SkippedCount=0
				for _, s := range got.Summaries {
					assert.Equal(t, 1, s.SignalCount)
					assert.Equal(t, 0, s.SkippedCount)
					// horizon=5,10,20 全て評価済み（30本あるので）
					for _, h := range signalPerformanceHorizons {
						st := s.Stats[h]
						assert.NotNil(t, st)
						assert.Equal(t, 1, st.EvaluatedCount)
					}
				}
			},
		},
		{
			name: "正常系: method 指定時は Signals 明細が非空",
			filter: &models.SignalPerformanceFilter{
				From:   from,
				To:     to,
				Method: method1,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Eq(&models.SignalPerformanceFilter{
						From:   from,
						To:     to,
						Method: method1,
					})).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignal("7203", method1, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(
						makePrices("7203", signalDate, 30, 1000), nil,
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				// method 指定時は Signals に明細が入る
				assert.NotEmpty(t, got.Signals)
				assert.Equal(t, method1, got.Signals[0].Method)
			},
		},
		{
			name: "正常系: method 未指定時は Signals が空スライス",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignal("7203", method1, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(
						makePrices("7203", signalDate, 30, 1000), nil,
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				assert.Equal(t, []*models.EvaluatedSignal{}, got.Signals)
			},
		},
		{
			name: "正常系: シグナル0件 → 空レスポンス / priceRepo は呼ばれない",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{}, nil)
					return m
				},
				// priceRepo の EXPECT は設定しない（呼ばれない）
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				assert.Equal(t, from, got.From)
				assert.Equal(t, to, got.To)
				assert.Empty(t, got.Summaries)
				assert.Equal(t, []*models.EvaluatedSignal{}, got.Signals)
			},
		},
		{
			name: "異常系: analyzeRepo エラー → エラー伝播",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
				},
			},
			wantErr: true,
		},
		{
			name: "異常系: priceRepo エラー → エラー伝播",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignal("7203", method1, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
					return m
				},
			},
			wantErr: true,
		},
		{
			name: "正常系: Sell シグナルのリターン符号反転",
			filter: &models.SignalPerformanceFilter{
				From:   from,
				To:     to,
				Method: method1,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignal("7203", method1, models.AnalyzeStockBrandPriceHistoryActionSell, signalDate),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					// 価格が上昇するリストを返す（Sell の場合はリターン符号が反転されて負になるはず）
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(
						makePrices("7203", signalDate, 30, 1000), nil,
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				assert.NotEmpty(t, got.Signals)
				sig := got.Signals[0]
				assert.Equal(t, models.AnalyzeStockBrandPriceHistoryActionSell, sig.Action)
				// 価格が上昇しているので Sell では全 horizon でリターンが負
				for _, h := range signalPerformanceHorizons {
					r := sig.Returns[h]
					assert.NotNil(t, r)
					assert.True(t, r.IsNegative(), "Sell時リターンは負であるべき (horizon=%d, r=%v)", h, r)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewSignalPerformanceInteractor(
				tt.fields.analyzeRepo(ctrl),
				tt.fields.priceRepo(ctrl),
			)
			got, err := interactor.GetSignalPerformance(context.Background(), tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// --- filterPricesFrom 単体テスト ---

func TestFilterPricesFrom(t *testing.T) {
	d := func(day int) time.Time { return spDate(2024, 1, day) }

	prices := []*models.StockBrandDailyPrice{
		spPrice("7203", d(3), 100),
		spPrice("7203", d(4), 101),
		spPrice("7203", d(5), 102),
		spPrice("7203", d(8), 103),
	}

	tests := []struct {
		name       string
		signalDate time.Time
		wantLen    int
		wantFirst  float64
	}{
		{
			name:       "シグナル日と一致する価格から返す",
			signalDate: d(4),
			wantLen:    3,
			wantFirst:  101,
		},
		{
			name:       "シグナル日以降の最初の価格から返す（営業日ずれ）",
			signalDate: d(6),
			wantLen:    1,
			wantFirst:  103,
		},
		{
			name:       "全価格より前のシグナル日 → 全件返す",
			signalDate: d(1),
			wantLen:    4,
			wantFirst:  100,
		},
		{
			name:       "全価格より後のシグナル日 → nil",
			signalDate: d(10),
			wantLen:    0,
			wantFirst:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterPricesFrom(prices, tt.signalDate)
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				v, _ := got[0].Adjclose.Float64()
				assert.InDelta(t, tt.wantFirst, v, 1e-9)
			}
		})
	}
}

// --- rankBands / scoreQuartiles テスト ---

func TestSignalPerformanceInteractor_BandAggregation(t *testing.T) {
	from := spDate(2024, 1, 4)
	to := spDate(2024, 3, 31)
	signalDate := spDate(2024, 1, 10)
	method1 := models.AnalyzeStockBrandPriceHistoryMethodFindMACDBullishV1

	rank1 := 1
	rank5 := 5
	score1 := decimal.NewFromFloat(0.8)
	score2 := decimal.NewFromFloat(0.3)

	// SignalRank / Score 付きシグナルを返すヘルパー
	spSignalWithBands := func(symbol string, rank *int, score *decimal.Decimal) *models.AnalyzeStockBrandPriceHistory {
		sig := spSignal(symbol, method1, models.AnalyzeStockBrandPriceHistoryActionBuy, signalDate)
		sig.SignalRank = rank
		sig.Score = score
		return sig
	}

	makePrices := func(symbol string) []*models.StockBrandDailyPrice {
		prices := make([]*models.StockBrandDailyPrice, 30)
		for i := 0; i < 30; i++ {
			prices[i] = spPrice(symbol, signalDate.AddDate(0, 0, i), 1000+float64(i))
		}
		return prices
	}

	type fields struct {
		analyzeRepo func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository
		priceRepo   func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository
	}

	tests := []struct {
		name   string
		filter *models.SignalPerformanceFilter
		fields fields
		check  func(t *testing.T, got *models.SignalPerformance)
	}{
		{
			name: "method 指定時は rankBands と scoreQuartiles が非nil",
			filter: &models.SignalPerformanceFilter{
				From:   from,
				To:     to,
				Method: method1,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignalWithBands("7203", &rank1, &score1),
						spSignalWithBands("6758", &rank5, &score2),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(
						append(makePrices("7203"), makePrices("6758")...), nil,
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				// method 指定時は rankBands が3帯固定で返る
				assert.NotNil(t, got.RankBands)
				assert.Len(t, got.RankBands, 3, "ランク帯は常に3帯")
				assert.Equal(t, "1-3", got.RankBands[0].Band)
				assert.Equal(t, "4-10", got.RankBands[1].Band)
				assert.Equal(t, "11+", got.RankBands[2].Band)
				// rank=1 は 1-3 帯, rank=5 は 4-10 帯
				assert.Equal(t, 1, got.RankBands[0].SignalCount)
				assert.Equal(t, 1, got.RankBands[1].SignalCount)
				assert.Equal(t, 0, got.RankBands[2].SignalCount)

				// score が2件のため n<4 → scoreQuartiles は空スライス
				assert.NotNil(t, got.ScoreQuartiles)
				assert.Empty(t, got.ScoreQuartiles, "n<4 のとき空スライス")
			},
		},
		{
			name: "method 未指定時は rankBands と scoreQuartiles が nil",
			filter: &models.SignalPerformanceFilter{
				From: from,
				To:   to,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						spSignalWithBands("7203", &rank1, &score1),
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(
						makePrices("7203"), nil,
					)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				// method 未指定時は rankBands/scoreQuartiles は nil
				assert.Nil(t, got.RankBands)
				assert.Nil(t, got.ScoreQuartiles)
			},
		},
		{
			name: "score 4件以上の場合 scoreQuartiles が4帯返る",
			filter: &models.SignalPerformanceFilter{
				From:   from,
				To:     to,
				Method: method1,
			},
			fields: fields{
				analyzeRepo: func(ctrl *gomock.Controller) *mock_repositories.MockAnalyzeStockBrandPriceHistoryRepository {
					m := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					s1 := decimal.NewFromFloat(0.1)
					s2 := decimal.NewFromFloat(0.3)
					s3 := decimal.NewFromFloat(0.6)
					s4 := decimal.NewFromFloat(0.9)
					m.EXPECT().FindByCreatedAtRange(gomock.Any(), gomock.Any()).Return([]*models.AnalyzeStockBrandPriceHistory{
						{ID: "1", TickerSymbol: "7203", Method: method1, Action: "Buy", CreatedAt: signalDate, Score: &s1},
						{ID: "2", TickerSymbol: "6758", Method: method1, Action: "Buy", CreatedAt: signalDate, Score: &s2},
						{ID: "3", TickerSymbol: "9984", Method: method1, Action: "Buy", CreatedAt: signalDate, Score: &s3},
						{ID: "4", TickerSymbol: "3382", Method: method1, Action: "Buy", CreatedAt: signalDate, Score: &s4},
					}, nil)
					return m
				},
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					var prices []*models.StockBrandDailyPrice
					for _, sym := range []string{"7203", "6758", "9984", "3382"} {
						prices = append(prices, makePrices(sym)...)
					}
					m.EXPECT().ListRangePricesBySymbols(gomock.Any(), gomock.Any()).Return(prices, nil)
					return m
				},
			},
			check: func(t *testing.T, got *models.SignalPerformance) {
				assert.NotNil(t, got.ScoreQuartiles)
				assert.Len(t, got.ScoreQuartiles, 4, "n=4 のとき4帯返る")
				assert.Equal(t, "Q1", got.ScoreQuartiles[0].Band)
				assert.Equal(t, "Q2", got.ScoreQuartiles[1].Band)
				assert.Equal(t, "Q3", got.ScoreQuartiles[2].Band)
				assert.Equal(t, "Q4", got.ScoreQuartiles[3].Band)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewSignalPerformanceInteractor(
				tt.fields.analyzeRepo(ctrl),
				tt.fields.priceRepo(ctrl),
			)
			got, err := interactor.GetSignalPerformance(context.Background(), tt.filter)
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

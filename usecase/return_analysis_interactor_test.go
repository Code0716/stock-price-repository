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

func TestReturnAnalysisInteractor_GetReturnAnalysis(t *testing.T) {
	// 銘柄は 00:00、日経は 09:00 と時刻を変えても日付（YYYY-MM-DD）で結合できることを確認する
	sd := func(day int) time.Time { return time.Date(2024, 1, day, 0, 0, 0, 0, time.UTC) }
	nd := func(day int) time.Time { return time.Date(2024, 1, day, 9, 0, 0, 0, time.UTC) }
	stockPrice := func(day int, adj float64) *models.StockBrandDailyPrice {
		return &models.StockBrandDailyPrice{TickerSymbol: "7203", Date: sd(day), Adjclose: decimal.NewFromFloat(adj)}
	}
	benchPrice := func(day int, adj float64) *models.IndexStockAverageDailyPrice {
		return &models.IndexStockAverageDailyPrice{Date: nd(day), Adjclose: decimal.NewFromFloat(adj)}
	}

	from := sd(1)
	to := sd(10)
	order := models.SortOrderAsc
	wantFilter := models.ListDailyPricesBySymbolFilter{
		TickerSymbol: "7203",
		DateFrom:     &from,
		DateTo:       &to,
		DateOrder:    &order,
	}

	type fields struct {
		stockRepo  func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository
		nikkeiRepo func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository
		topixRepo  func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository
	}
	tests := []struct {
		name      string
		fields    fields
		benchmark string
		wantErr   bool
		check     func(t *testing.T, got *models.ReturnAnalysis)
	}{
		{
			name:      "正常系: 銘柄と日経を日付結合して指標を算出（重複しない日は除外）",
			benchmark: models.BenchmarkNikkei,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return([]*models.StockBrandDailyPrice{
						stockPrice(4, 100), stockPrice(5, 110), stockPrice(6, 120), stockPrice(9, 121), stockPrice(10, 130),
					}, nil)
					return m
				},
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					m := mock_repositories.NewMockNikkeiRepository(ctrl)
					// 1/10 は欠落 → 結合後は 4 営業日
					m.EXPECT().ListNikkeiStockAverageDailyPrices(gomock.Any(), &from, &to).Return(models.IndexStockAverageDailyPrices{
						benchPrice(4, 1000), benchPrice(5, 1010), benchPrice(6, 1020), benchPrice(9, 1015),
					}, nil)
					return m
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					return mock_repositories.NewMockTopixRepository(ctrl)
				},
			},
			check: func(t *testing.T, got *models.ReturnAnalysis) {
				assert.Equal(t, "7203", got.Symbol)
				assert.Equal(t, models.BenchmarkNikkei, got.Benchmark)
				assert.Equal(t, 4, got.TradingDays)
				assert.Equal(t, "2024-01-04", got.From)
				assert.Equal(t, "2024-01-09", got.To)
				// 銘柄 121/100-1=0.21, 日経 1015/1000-1=0.015, 超過=0.195
				cr, _ := got.CumulativeReturn.Float64()
				br, _ := got.BenchmarkReturn.Float64()
				er, _ := got.ExcessReturn.Float64()
				assert.InDelta(t, 0.21, cr, 1e-9)
				assert.InDelta(t, 0.015, br, 1e-9)
				assert.InDelta(t, 0.195, er, 1e-9)
				// 相関・βなどは算出され 0 以外（厳密値ではなく非ゼロを確認）
				assert.False(t, got.Correlation.IsZero())
			},
		},
		{
			name:      "正常系: benchmark=topix で TopixRepository が呼ばれ Benchmark=topix",
			benchmark: models.BenchmarkTopix,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return([]*models.StockBrandDailyPrice{
						stockPrice(4, 100), stockPrice(5, 110), stockPrice(6, 120), stockPrice(9, 121), stockPrice(10, 130),
					}, nil)
					return m
				},
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					// topix 指定時は nikkeiRepo は呼ばれない
					return mock_repositories.NewMockNikkeiRepository(ctrl)
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					m := mock_repositories.NewMockTopixRepository(ctrl)
					m.EXPECT().ListTopixDailyPrices(gomock.Any(), &from, &to).Return(models.IndexStockAverageDailyPrices{
						benchPrice(4, 2000), benchPrice(5, 2020), benchPrice(6, 2040), benchPrice(9, 2030), benchPrice(10, 2060),
					}, nil)
					return m
				},
			},
			check: func(t *testing.T, got *models.ReturnAnalysis) {
				assert.Equal(t, "7203", got.Symbol)
				assert.Equal(t, models.BenchmarkTopix, got.Benchmark)
				assert.Equal(t, 5, got.TradingDays)
				// 銘柄 130/100-1=0.30, TOPIX 2060/2000-1=0.03
				cr, _ := got.CumulativeReturn.Float64()
				br, _ := got.BenchmarkReturn.Float64()
				assert.InDelta(t, 0.30, cr, 1e-9)
				assert.InDelta(t, 0.03, br, 1e-9)
			},
		},
		{
			name:      "データ不足: 結合後1日のみ → 指標はゼロ値",
			benchmark: models.BenchmarkNikkei,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return([]*models.StockBrandDailyPrice{
						stockPrice(4, 100), stockPrice(5, 110),
					}, nil)
					return m
				},
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					m := mock_repositories.NewMockNikkeiRepository(ctrl)
					m.EXPECT().ListNikkeiStockAverageDailyPrices(gomock.Any(), &from, &to).Return(models.IndexStockAverageDailyPrices{
						benchPrice(4, 1000),
					}, nil)
					return m
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					return mock_repositories.NewMockTopixRepository(ctrl)
				},
			},
			check: func(t *testing.T, got *models.ReturnAnalysis) {
				assert.Equal(t, 1, got.TradingDays)
				assert.True(t, got.CumulativeReturn.IsZero())
				assert.True(t, got.SharpeRatio.IsZero())
				assert.Equal(t, "2024-01-04", got.From)
				assert.Equal(t, "2024-01-04", got.To)
			},
		},
		{
			name:      "異常系: 銘柄日足取得でエラー",
			benchmark: models.BenchmarkNikkei,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return(nil, errors.New("db error"))
					return m
				},
				// 日経は呼ばれない
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					return mock_repositories.NewMockNikkeiRepository(ctrl)
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					return mock_repositories.NewMockTopixRepository(ctrl)
				},
			},
			wantErr: true,
		},
		{
			name:      "異常系: 日経取得でエラー",
			benchmark: models.BenchmarkNikkei,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return([]*models.StockBrandDailyPrice{
						stockPrice(4, 100), stockPrice(5, 110),
					}, nil)
					return m
				},
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					m := mock_repositories.NewMockNikkeiRepository(ctrl)
					m.EXPECT().ListNikkeiStockAverageDailyPrices(gomock.Any(), &from, &to).Return(nil, errors.New("db error"))
					return m
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					return mock_repositories.NewMockTopixRepository(ctrl)
				},
			},
			wantErr: true,
		},
		{
			name:      "異常系: TOPIX取得でエラー",
			benchmark: models.BenchmarkTopix,
			fields: fields{
				stockRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().ListDailyPricesBySymbol(gomock.Any(), wantFilter).Return([]*models.StockBrandDailyPrice{
						stockPrice(4, 100), stockPrice(5, 110),
					}, nil)
					return m
				},
				nikkeiRepo: func(ctrl *gomock.Controller) *mock_repositories.MockNikkeiRepository {
					return mock_repositories.NewMockNikkeiRepository(ctrl)
				},
				topixRepo: func(ctrl *gomock.Controller) *mock_repositories.MockTopixRepository {
					m := mock_repositories.NewMockTopixRepository(ctrl)
					m.EXPECT().ListTopixDailyPrices(gomock.Any(), &from, &to).Return(nil, errors.New("db error"))
					return m
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewReturnAnalysisInteractor(
				tt.fields.stockRepo(ctrl),
				tt.fields.nikkeiRepo(ctrl),
				tt.fields.topixRepo(ctrl),
			)
			got, err := interactor.GetReturnAnalysis(context.Background(), "7203", &from, &to, tt.benchmark)

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

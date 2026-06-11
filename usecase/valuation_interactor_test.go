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

// --- computeValuation 単体テスト ---

func TestComputeValuation(t *testing.T) {
	dec := func(f float64) *decimal.Decimal { v := decimal.NewFromFloat(f); return &v }

	tests := []struct {
		name        string
		close       float64
		trailingEPS *decimal.Decimal
		forecastEPS *decimal.Decimal
		bps         *decimal.Decimal
		forecastDPS *decimal.Decimal
		wantPER     *float64
		wantFwdPER  *float64
		wantPBR     *float64
		wantROE     *float64
		wantDivYld  *float64
	}{
		{
			name:        "正常系: 全指標算出",
			close:       1000,
			trailingEPS: dec(100),
			forecastEPS: dec(125),
			bps:         dec(500),
			forecastDPS: dec(20),
			wantPER:     ptr(10.0),
			wantFwdPER:  ptr(8.0),
			wantPBR:     ptr(2.0),
			wantROE:     ptr(0.2),
			wantDivYld:  ptr(0.02), // 20 / 1000
		},
		{
			name:        "配当利回り: DPS=50, Close=2500 → 0.02",
			close:       2500,
			trailingEPS: dec(100),
			forecastEPS: dec(125),
			bps:         dec(500),
			forecastDPS: dec(50),
			wantPER:     ptr(25.0),
			wantFwdPER:  ptr(20.0),
			wantPBR:     ptr(5.0),
			wantROE:     ptr(0.2),
			wantDivYld:  ptr(0.02), // 50 / 2500
		},
		{
			name:        "DPS=nil: ForecastDividendYield=nil",
			close:       1000,
			trailingEPS: dec(100),
			forecastEPS: dec(125),
			bps:         dec(500),
			forecastDPS: nil,
			wantPER:     ptr(10.0),
			wantFwdPER:  ptr(8.0),
			wantPBR:     ptr(2.0),
			wantROE:     ptr(0.2),
			wantDivYld:  nil,
		},
		{
			name:        "赤字EPS: PER=nil, ROE は負で算出",
			close:       1000,
			trailingEPS: dec(-50),
			forecastEPS: dec(30),
			bps:         dec(500),
			forecastDPS: dec(10),
			wantPER:     nil,
			wantFwdPER:  ptr(1000.0 / 30),
			wantPBR:     ptr(2.0),
			wantROE:     ptr(-0.1),
			wantDivYld:  ptr(0.01),
		},
		{
			name:        "EPS=0: PER=nil, ROE=0 (ゼロEPS/BPSは0算出)",
			close:       1000,
			trailingEPS: dec(0),
			forecastEPS: dec(0),
			bps:         dec(500),
			forecastDPS: nil,
			wantPER:     nil,
			wantFwdPER:  nil,
			wantPBR:     ptr(2.0),
			wantROE:     ptr(0.0), // trailingEPS=0, bps>0 → ROE=0
			wantDivYld:  nil,
		},
		{
			name:        "BPS=0: PBR/ROE=nil",
			close:       1000,
			trailingEPS: dec(100),
			forecastEPS: dec(120),
			bps:         dec(0),
			forecastDPS: nil,
			wantPER:     ptr(10.0),
			wantFwdPER:  ptr(1000.0 / 120),
			wantPBR:     nil,
			wantROE:     nil,
			wantDivYld:  nil,
		},
		{
			name:        "EPS/BPS=nil: PER/ROE=nil",
			close:       1000,
			trailingEPS: nil,
			forecastEPS: nil,
			bps:         nil,
			forecastDPS: nil,
			wantPER:     nil,
			wantFwdPER:  nil,
			wantPBR:     nil,
			wantROE:     nil,
			wantDivYld:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			close := decimal.NewFromFloat(tt.close)
			got := computeValuation(close, tt.trailingEPS, tt.forecastEPS, tt.bps, tt.forecastDPS)

			assertDecPtr(t, "PER", tt.wantPER, got.PER)
			assertDecPtr(t, "ForwardPER", tt.wantFwdPER, got.ForwardPER)
			assertDecPtr(t, "PBR", tt.wantPBR, got.PBR)
			assertDecPtr(t, "ROE", tt.wantROE, got.ROE)
			assertDecPtr(t, "ForecastDividendYield", tt.wantDivYld, got.ForecastDividendYield)
		})
	}
}

func ptr(f float64) *float64 { return &f }

func assertDecPtr(t *testing.T, name string, want *float64, got *decimal.Decimal) {
	t.Helper()
	if want == nil {
		assert.Nil(t, got, name+" should be nil")
		return
	}
	if assert.NotNil(t, got, name+" should not be nil") {
		f, _ := got.Float64()
		assert.InDelta(t, *want, f, 1e-6, name)
	}
}

// --- GetValuation usecase テスト ---

func TestValuationInteractor_GetValuation(t *testing.T) {
	now := time.Now()
	fyEnd := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	makePrice := func(close float64) *models.StockBrandDailyPrice {
		return &models.StockBrandDailyPrice{
			TickerSymbol: "7203",
			Date:         now,
			Close:        decimal.NewFromFloat(close),
		}
	}

	eps := decimal.NewFromFloat(100)
	forecastEPS := decimal.NewFromFloat(125)
	bps := decimal.NewFromFloat(500)
	dps := decimal.NewFromFloat(50)

	makeStmts := func(period string, hasEPS, hasBPS, hasFEPS, hasDPS bool) []*models.FinStatement {
		s := &models.FinStatement{
			TickerSymbol:        "7203",
			DisclosedDate:       now,
			TypeOfCurrentPeriod: period,
			FiscalYearEnd:       &fyEnd,
		}
		if hasEPS {
			s.EarningsPerShare = &eps
		}
		if hasBPS {
			s.BookValuePerShare = &bps
		}
		if hasFEPS {
			s.ForecastEPS = &forecastEPS
		}
		if hasDPS {
			s.ForecastDividendPerShareAnnual = &dps
		}
		return []*models.FinStatement{s}
	}

	type fields struct {
		finRepo   func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository
		priceRepo func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		check   func(t *testing.T, got *models.Valuation)
	}{
		{
			name: "正常系: FY決算でPER/PBR/ROE/予想PER/予想配当利回り全算出",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(makePrice(2500), nil)
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().FindBySymbol(gomock.Any(), &models.FinStatementFilter{TickerSymbol: "7203", Limit: 12}).
						Return(makeStmts("FY", true, true, true, true), nil)
					return m
				},
			},
			check: func(t *testing.T, got *models.Valuation) {
				assert.Equal(t, "7203", got.Symbol)
				assert.NotNil(t, got.Close)
				assertDecPtr(t, "PER", ptr(25.0), got.PER)       // 2500/100
				assertDecPtr(t, "ForwardPER", ptr(20.0), got.ForwardPER) // 2500/125
				assertDecPtr(t, "PBR", ptr(5.0), got.PBR)        // 2500/500
				assertDecPtr(t, "ROE", ptr(0.2), got.ROE)         // 100/500
				assertDecPtr(t, "ForecastDividendYield", ptr(0.02), got.ForecastDividendYield) // 50/2500
				assert.Equal(t, "2025-03", got.FiscalPeriod)
			},
		},
		{
			name: "DPS=nil: ForecastDividendYield=nil",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(makePrice(1000), nil)
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().FindBySymbol(gomock.Any(), gomock.Any()).
						Return(makeStmts("FY", true, true, true, false), nil) // DPS なし
					return m
				},
			},
			check: func(t *testing.T, got *models.Valuation) {
				assert.Nil(t, got.ForecastDividendYield)
				assert.Nil(t, got.ForecastDividendPerShareAnnual)
				assert.NotNil(t, got.PER)
			},
		},
		{
			name: "四半期決算のみ: trailingEPS=nil, PER/ROE=nil",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(makePrice(1000), nil)
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().FindBySymbol(gomock.Any(), gomock.Any()).
						Return(makeStmts("1Q", true, true, true, false), nil) // 1Q = 四半期
					return m
				},
			},
			check: func(t *testing.T, got *models.Valuation) {
				assert.Nil(t, got.PER)
				assert.Nil(t, got.ROE)
				assert.NotNil(t, got.ForwardPER) // forecastEPS は期種別問わず取得
				assert.NotNil(t, got.PBR)        // bps は期種別問わず取得
				assert.Equal(t, "", got.FiscalPeriod)
			},
		},
		{
			name: "財務データ空: 指標は全nil、Closeは設定",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(makePrice(1000), nil)
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().FindBySymbol(gomock.Any(), gomock.Any()).Return([]*models.FinStatement{}, nil)
					return m
				},
			},
			check: func(t *testing.T, got *models.Valuation) {
				assert.NotNil(t, got.Close)
				assert.Nil(t, got.PER)
				assert.Nil(t, got.PBR)
				assert.Nil(t, got.ROE)
				assert.Nil(t, got.ForecastDividendYield)
			},
		},
		{
			name: "異常系: 価格取得エラー",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(nil, errors.New("db error"))
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					return mock_repositories.NewMockFinStatementRepository(ctrl)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewValuationInteractor(tt.fields.finRepo(ctrl), tt.fields.priceRepo(ctrl))
			got, err := interactor.GetValuation(context.Background(), "7203")

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

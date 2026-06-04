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
		wantPER     *float64
		wantFwdPER  *float64
		wantPBR     *float64
		wantROE     *float64
	}{
		{
			name:        "正常系: 全指標算出",
			close:       1000,
			trailingEPS: dec(100),
			forecastEPS: dec(125),
			bps:         dec(500),
			wantPER:     ptr(10.0),
			wantFwdPER:  ptr(8.0),
			wantPBR:     ptr(2.0),
			wantROE:     ptr(0.2),
		},
		{
			name:        "赤字EPS: PER=nil, ROE は負で算出",
			close:       1000,
			trailingEPS: dec(-50),
			forecastEPS: dec(30),
			bps:         dec(500),
			wantPER:     nil,
			wantFwdPER:  ptr(1000.0 / 30),
			wantPBR:     ptr(2.0),
			wantROE:     ptr(-0.1),
		},
		{
			name:        "EPS=0: PER=nil, ROE=0 (ゼロEPS/BPSは0算出)",
			close:       1000,
			trailingEPS: dec(0),
			forecastEPS: dec(0),
			bps:         dec(500),
			wantPER:     nil,
			wantFwdPER:  nil,
			wantPBR:     ptr(2.0),
			wantROE:     ptr(0.0), // trailingEPS=0, bps>0 → ROE=0 (EPS=0は赤字でないため算出する)
		},
		{
			name:        "BPS=0: PBR/ROE=nil",
			close:       1000,
			trailingEPS: dec(100),
			forecastEPS: dec(120),
			bps:         dec(0),
			wantPER:     ptr(10.0),
			wantFwdPER:  ptr(1000.0 / 120),
			wantPBR:     nil,
			wantROE:     nil,
		},
		{
			name:        "EPS/BPS=nil: PER/ROE=nil",
			close:       1000,
			trailingEPS: nil,
			forecastEPS: nil,
			bps:         nil,
			wantPER:     nil,
			wantFwdPER:  nil,
			wantPBR:     nil,
			wantROE:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			close := decimal.NewFromFloat(tt.close)
			got := computeValuation(close, tt.trailingEPS, tt.forecastEPS, tt.bps)

			assertDecPtr(t, "PER", tt.wantPER, got.PER)
			assertDecPtr(t, "ForwardPER", tt.wantFwdPER, got.ForwardPER)
			assertDecPtr(t, "PBR", tt.wantPBR, got.PBR)
			assertDecPtr(t, "ROE", tt.wantROE, got.ROE)
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

	makeStmts := func(period string, hasEPS, hasBPS, hasFEPS bool) []*models.FinStatement {
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
			name: "正常系: FY決算でPER/PBR/ROE/予想PER全算出",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) *mock_repositories.MockStockBrandsDailyPriceRepository {
					m := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					m.EXPECT().GetLatestPriceBySymbol(gomock.Any(), "7203").Return(makePrice(1000), nil)
					return m
				},
				finRepo: func(ctrl *gomock.Controller) *mock_repositories.MockFinStatementRepository {
					m := mock_repositories.NewMockFinStatementRepository(ctrl)
					m.EXPECT().FindBySymbol(gomock.Any(), &models.FinStatementFilter{TickerSymbol: "7203", Limit: 12}).
						Return(makeStmts("FY", true, true, true), nil)
					return m
				},
			},
			check: func(t *testing.T, got *models.Valuation) {
				assert.Equal(t, "7203", got.Symbol)
				assert.NotNil(t, got.Close)
				assertDecPtr(t, "PER", ptr(10.0), got.PER)
				assertDecPtr(t, "ForwardPER", ptr(8.0), got.ForwardPER)
				assertDecPtr(t, "PBR", ptr(2.0), got.PBR)
				assertDecPtr(t, "ROE", ptr(0.2), got.ROE)
				assert.Equal(t, "2025-03", got.FiscalPeriod)
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
						Return(makeStmts("1Q", true, true, true), nil) // 1Q = 四半期
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

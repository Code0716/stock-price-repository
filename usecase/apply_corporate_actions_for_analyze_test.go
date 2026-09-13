package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func Test_stockBrandsDailyStockPriceInteractorImpl_applyCorporateActionsForAnalyze(t *testing.T) {
	now := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	effectiveDate := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)

	type fields struct {
		tx                                          func(ctrl *gomock.Controller) repositories.Transaction
		stockBrandsDailyPriceForAnalyzeRepository   func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		appliedStockSplitsHistoryRepository         func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository
		appliedStockConsolidationsHistoryRepository func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository
		stockAPIClient                              func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	tests := []struct {
		name   string
		fields fields
		prices []*gateway.StockPrice
	}{
		{
			name: "AdjustmentFactorが1の銘柄は何もしない",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl) // 呼ばれない
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl) // 呼ばれない
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					return mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl) // 呼ばれない
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl) // 呼ばれない
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					return mock_gateway.NewMockStockAPIClient(ctrl) // 呼ばれない
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "1301", Date: now, AdjustmentFactor: decimal.NewFromInt(1)},
			},
		},
		{
			name: "AdjustmentFactorが未設定(ゼロ値)の銘柄は何もしない",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					return mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					return mock_gateway.NewMockStockAPIClient(ctrl)
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "1301", Date: now},
			},
		},
		{
			name: "分割(Factor<1)を検知した銘柄は全期間を再取得しfor_analyzeを上書き、splits_historyに正しい日付で記録する",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					}).Times(1)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, prices []*models.StockBrandDailyPriceForAnalyze) error {
							if len(prices) != 1 {
								t.Errorf("expected 1 price, got %d", len(prices))
							}
							return nil
						},
					).Times(1)
					return mock
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "7678", effectiveDate).Return(false, nil).Times(1)
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, h *models.AppliedStockSplitHistory) error {
							if h.Symbol != "7678" {
								t.Errorf("Symbol = %s, want 7678", h.Symbol)
							}
							if !h.SplitDate.Equal(effectiveDate) {
								t.Errorf("SplitDate = %v, want %v", h.SplitDate, effectiveDate)
							}
							// AdjustmentFactor=0.5 -> ratio(旧/新) = 2.0
							if !h.Ratio.Equal(decimal.NewFromInt(2)) {
								t.Errorf("Ratio = %s, want 2", h.Ratio.String())
							}
							return nil
						},
					).Times(1)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl) // 呼ばれない
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().
						GetDailyPricesBySymbolAndRange(gomock.Any(), gateway.StockAPISymbol("7678"), gomock.Any(), gomock.Any()).
						Return([]*gateway.StockPrice{
							{
								TickerSymbol:    "7678",
								Date:            effectiveDate,
								Open:            decimal.NewFromInt(3005),
								High:            decimal.NewFromInt(3065),
								Low:             decimal.NewFromInt(2931),
								Close:           decimal.NewFromInt(3035),
								Volume:          86900,
								AdjustmentClose: decimal.NewFromInt(3035),
							},
						}, nil).Times(1)
					return mock
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "7678", Date: effectiveDate, AdjustmentFactor: decimal.NewFromFloat(0.5)},
			},
		},
		{
			name: "既に適用済み(splits_historyに存在)の場合は再取得しない",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl) // 呼ばれない
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl) // 呼ばれない
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "7678", effectiveDate).Return(true, nil).Times(1)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					return mock_gateway.NewMockStockAPIClient(ctrl) // 呼ばれない
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "7678", Date: effectiveDate, AdjustmentFactor: decimal.NewFromFloat(0.5)},
			},
		},
		{
			name: "併合(Factor>1)を検知した銘柄はconsolidations_historyに記録する",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					}).Times(1)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().CreateStockBrandDailyPriceForAnalyze(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					return mock
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					return mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl) // 呼ばれない
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "9999", effectiveDate).Return(false, nil).Times(1)
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, h *models.AppliedStockConsolidationHistory) error {
							if !h.Ratio.Equal(decimal.NewFromInt(5)) {
								t.Errorf("Ratio = %s, want 5", h.Ratio.String())
							}
							return nil
						},
					).Times(1)
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().
						GetDailyPricesBySymbolAndRange(gomock.Any(), gateway.StockAPISymbol("9999"), gomock.Any(), gomock.Any()).
						Return([]*gateway.StockPrice{
							{TickerSymbol: "9999", Date: effectiveDate, Open: decimal.NewFromInt(500), High: decimal.NewFromInt(510), Low: decimal.NewFromInt(490), Close: decimal.NewFromInt(500), Volume: 100},
						}, nil).Times(1)
					return mock
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "9999", Date: effectiveDate, AdjustmentFactor: decimal.NewFromInt(5)},
			},
		},
		{
			name: "APIエラーは1銘柄の失敗として無視し処理を継続する(呼び出し全体はエラーを返さない)",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl) // 呼ばれない
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					return mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl) // 呼ばれない
				},
				appliedStockSplitsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockSplitsHistoryRepository {
					mock := mock_repositories.NewMockAppliedStockSplitsHistoryRepository(ctrl)
					mock.EXPECT().Exists(gomock.Any(), "7678", effectiveDate).Return(false, nil).Times(1)
					return mock
				},
				appliedStockConsolidationsHistoryRepository: func(ctrl *gomock.Controller) repositories.AppliedStockConsolidationsHistoryRepository {
					return mock_repositories.NewMockAppliedStockConsolidationsHistoryRepository(ctrl)
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().
						GetDailyPricesBySymbolAndRange(gomock.Any(), gateway.StockAPISymbol("7678"), gomock.Any(), gomock.Any()).
						Return(nil, errors.New("j-quants api error")).Times(1)
					return mock
				},
			},
			prices: []*gateway.StockPrice{
				{TickerSymbol: "7678", Date: effectiveDate, AdjustmentFactor: decimal.NewFromFloat(0.5)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			si := &stockBrandsDailyStockPriceInteractorImpl{
				tx: tt.fields.tx(ctrl),
				stockBrandsDailyPriceForAnalyzeRepository:   tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl),
				appliedStockSplitsHistoryRepository:         tt.fields.appliedStockSplitsHistoryRepository(ctrl),
				appliedStockConsolidationsHistoryRepository: tt.fields.appliedStockConsolidationsHistoryRepository(ctrl),
				stockAPIClient: tt.fields.stockAPIClient(ctrl),
			}

			if err := si.applyCorporateActionsForAnalyze(context.Background(), tt.prices, now); err != nil {
				t.Errorf("applyCorporateActionsForAnalyze() error = %v, want nil (1銘柄の失敗で全体を止めない設計)", err)
			}
		})
	}
}

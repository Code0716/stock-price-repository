package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func dailyPickTestDates(now time.Time) []time.Time {
	dates := make([]time.Time, dailyStockPickWindowDays)
	for i := range dates {
		dates[i] = now.AddDate(0, 0, -i)
	}
	return dates
}

func TestCreateDailyStockPicksInteractorImpl_CreateDailyStockPicks(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)

	type fields struct {
		priceRepo  func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		brandRepo  func(ctrl *gomock.Controller) repositories.StockBrandRepository
		pickRepo   func(ctrl *gomock.Controller) repositories.DailyStockPickRepository
		slackAPI   func(ctrl *gomock.Controller) gateway.SlackAPIClientRaw
		notifyRepo func(ctrl *gomock.Controller) repositories.NotificationHistoryRepository
		tx         func(ctrl *gomock.Controller) repositories.Transaction
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "営業日がウォームアップ日数未満なら何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).
						Return([]time.Time{now}, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				pickRepo: func(ctrl *gomock.Controller) repositories.DailyStockPickRepository {
					return mock_repositories.NewMockDailyStockPickRepository(ctrl)
				},
				slackAPI: func(ctrl *gomock.Controller) gateway.SlackAPIClientRaw {
					return mock_gateway.NewMockSlackAPIClientRaw(ctrl)
				},
				notifyRepo: func(ctrl *gomock.Controller) repositories.NotificationHistoryRepository {
					return mock_repositories.NewMockNotificationHistoryRepository(ctrl)
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "当日分が作成済みかつ全件通知済みなら何もしない",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					dates := dailyPickTestDates(now)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				pickRepo: func(ctrl *gomock.Controller) repositories.DailyStockPickRepository {
					mock := mock_repositories.NewMockDailyStockPickRepository(ctrl)
					notifiedAt := now
					mock.EXPECT().ListByPickDate(gomock.Any(), now).Return([]*models.DailyStockPick{
						{TickerSymbol: "1000", NotifiedAt: &notifiedAt},
					}, nil)
					return mock
				},
				slackAPI: func(ctrl *gomock.Controller) gateway.SlackAPIClientRaw {
					return mock_gateway.NewMockSlackAPIClientRaw(ctrl)
				},
				notifyRepo: func(ctrl *gomock.Controller) repositories.NotificationHistoryRepository {
					return mock_repositories.NewMockNotificationHistoryRepository(ctrl)
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "当日分が作成済みだが未通知なら再通知だけ行う",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					dates := dailyPickTestDates(now)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					// FindAllMainMarkets は呼ばれない（再スクリーニングしない）
					return mock_repositories.NewMockStockBrandRepository(ctrl)
				},
				pickRepo: func(ctrl *gomock.Controller) repositories.DailyStockPickRepository {
					mock := mock_repositories.NewMockDailyStockPickRepository(ctrl)
					mock.EXPECT().ListByPickDate(gomock.Any(), now).Return([]*models.DailyStockPick{
						{PickDate: now, TickerSymbol: "1000", PickRank: 1, Score: decimal.NewFromInt(80)},
					}, nil)
					mock.EXPECT().MarkNotified(gomock.Any(), now, gomock.Any()).Return(nil)
					return mock
				},
				slackAPI: func(ctrl *gomock.Controller) gateway.SlackAPIClientRaw {
					mock := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
					mock.EXPECT().
						SendMessageByStrings(gomock.Any(), gateway.SlackChannelNameExchangeStockInfo, gomock.Any(), gomock.Any(), (*string)(nil)).
						Return("1234.5678", nil)
					return mock
				},
				notifyRepo: func(ctrl *gomock.Controller) repositories.NotificationHistoryRepository {
					mock := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
					mock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
		{
			name: "正常系: スクリーニング→保存→Slack通知→MarkNotifiedの順で実行される",
			fields: fields{
				priceRepo: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					dates := dailyPickTestDates(now)
					mock.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)
					mock.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
					return mock
				},
				brandRepo: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{}, nil)
					return mock
				},
				pickRepo: func(ctrl *gomock.Controller) repositories.DailyStockPickRepository {
					mock := mock_repositories.NewMockDailyStockPickRepository(ctrl)
					mock.EXPECT().ListByPickDate(gomock.Any(), now).Return(nil, nil)
					return mock
				},
				slackAPI: func(ctrl *gomock.Controller) gateway.SlackAPIClientRaw {
					return mock_gateway.NewMockSlackAPIClientRaw(ctrl)
				},
				notifyRepo: func(ctrl *gomock.Controller) repositories.NotificationHistoryRepository {
					return mock_repositories.NewMockNotificationHistoryRepository(ctrl)
				},
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					return mock_repositories.NewMockTransaction(ctrl)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interactor := NewCreateDailyStockPicksInteractor(
				tt.fields.tx(ctrl),
				tt.fields.priceRepo(ctrl),
				tt.fields.brandRepo(ctrl),
				tt.fields.pickRepo(ctrl),
				tt.fields.slackAPI(ctrl),
				tt.fields.notifyRepo(ctrl),
			)
			err := interactor.CreateDailyStockPicks(context.Background(), now, 0, -1, 1, false)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDailyStockPicksInteractorImpl_CreateDailyStockPicks_SaveAndNotify(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	dates := dailyPickTestDates(now)
	pickDate := dates[0]
	from := dates[len(dates)-1]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
	priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)

	brand := &models.StockBrand{ID: "brand-1", TickerSymbol: "1000", Name: "テスト銘柄"}
	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return([]*models.StockBrand{brand}, nil)

	prices := makeDailyPickUsecasePrices(dailyStockPickWindowDays, decimal.NewFromInt(1000), decimal.NewFromInt(2000))
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), models.ListDailyPricesBySymbolFilter{
		TickerSymbol: "1000",
		DateFrom:     &from,
		DateTo:       &pickDate,
		DateOrder:    dateOrderPtr(models.SortOrderAsc),
	}).Return(prices, nil)

	pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
	pickRepo.EXPECT().ListByPickDate(gomock.Any(), pickDate).Return(nil, nil)

	gomock.InOrder(
		pickRepo.EXPECT().DeleteByPickDate(gomock.Any(), pickDate).Return(nil),
		pickRepo.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, picks []*models.DailyStockPick) error {
				assert.Len(t, picks, 1)
				assert.Equal(t, "1000", picks[0].TickerSymbol)
				return nil
			}),
	)

	slackAPI := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	slackAPI.EXPECT().
		SendMessageByStrings(gomock.Any(), gateway.SlackChannelNameExchangeStockInfo, gomock.Any(), gomock.Any(), (*string)(nil)).
		Return("1234.5678", nil)

	notifyRepo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
	notifyRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	pickRepo.EXPECT().MarkNotified(gomock.Any(), pickDate, gomock.Any()).Return(nil)

	tx := mock_repositories.NewMockTransaction(ctrl)
	tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	interactor := NewCreateDailyStockPicksInteractor(tx, priceRepo, brandRepo, pickRepo, slackAPI, notifyRepo)
	err := interactor.CreateDailyStockPicks(context.Background(), now, 25, 4, 1, false)
	assert.NoError(t, err)
}

// TestCreateDailyStockPicksInteractorImpl_notify_RecordsOnceAcrossChunks
// 本文が複数チャンクに分割送信されても notification_history へは結合済み全文で1件だけ記録されることを確認する。
func TestCreateDailyStockPicksInteractorImpl_notify_RecordsOnceAcrossChunks(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	dates := dailyPickTestDates(now)
	pickDate := dates[0]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
	priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)

	brands := make([]*models.StockBrand, dailyStockPickDefaultTopN)
	for i := range brands {
		symbol := fmt.Sprintf("%04d", 1000+i)
		brands[i] = &models.StockBrand{ID: "brand-" + symbol, TickerSymbol: symbol, Name: "テスト銘柄" + symbol}
	}
	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)
	brandRepo.EXPECT().FindAllMainMarkets(gomock.Any()).Return(brands, nil)

	prices := makeDailyPickUsecasePrices(dailyStockPickWindowDays, decimal.NewFromInt(1000), decimal.NewFromInt(2000))
	priceRepo.EXPECT().ListDailyPricesBySymbol(gomock.Any(), gomock.Any()).Return(prices, nil).AnyTimes()

	pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
	pickRepo.EXPECT().ListByPickDate(gomock.Any(), pickDate).Return(nil, nil)
	pickRepo.EXPECT().DeleteByPickDate(gomock.Any(), pickDate).Return(nil)
	pickRepo.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).Return(nil)
	pickRepo.EXPECT().MarkNotified(gomock.Any(), pickDate, gomock.Any()).Return(nil)

	// 3500 rune の DailyPickSlackMaxRunes を超える本文になるよう、十分な銘柄数で複数チャンク送信を強制する。
	slackAPI := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	var chunkCount int
	slackAPI.EXPECT().
		SendMessageByStrings(gomock.Any(), gateway.SlackChannelNameExchangeStockInfo, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, _ string, _, _ *string) (string, error) {
			chunkCount++
			return "1234.5678", nil
		}).
		MinTimes(2)

	notifyRepo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)
	notifyRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	tx := mock_repositories.NewMockTransaction(ctrl)
	tx.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	interactor := NewCreateDailyStockPicksInteractor(tx, priceRepo, brandRepo, pickRepo, slackAPI, notifyRepo)
	err := interactor.CreateDailyStockPicks(context.Background(), now, dailyStockPickDefaultTopN, -1, 1, false)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, chunkCount, 2, "本文が複数チャンクに分割送信されていることの前提")
}

func TestCreateDailyStockPicksInteractorImpl_CreateDailyStockPicks_SlackFailureDoesNotMarkNotified(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	dates := dailyPickTestDates(now)
	pickDate := dates[0]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	priceRepo := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
	priceRepo.EXPECT().ListRecentTradingDates(gomock.Any(), now, dailyStockPickWindowDays).Return(dates, nil)

	// 既存あり・未通知の再通知パスなので FindAllMainMarkets（再スクリーニング）は呼ばれない
	brandRepo := mock_repositories.NewMockStockBrandRepository(ctrl)

	pickRepo := mock_repositories.NewMockDailyStockPickRepository(ctrl)
	pickRepo.EXPECT().ListByPickDate(gomock.Any(), pickDate).Return([]*models.DailyStockPick{
		{PickDate: pickDate, TickerSymbol: "1000", PickRank: 1, Score: decimal.NewFromInt(80)},
	}, nil)
	// MarkNotified は呼ばれない

	slackAPI := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
	slackAPI.EXPECT().
		SendMessageByStrings(gomock.Any(), gateway.SlackChannelNameExchangeStockInfo, gomock.Any(), gomock.Any(), (*string)(nil)).
		Return("", assert.AnError)

	// Slack送信が失敗するため notification_history への記録は行われない
	notifyRepo := mock_repositories.NewMockNotificationHistoryRepository(ctrl)

	tx := mock_repositories.NewMockTransaction(ctrl)

	interactor := NewCreateDailyStockPicksInteractor(tx, priceRepo, brandRepo, pickRepo, slackAPI, notifyRepo)
	err := interactor.CreateDailyStockPicks(context.Background(), now, 25, 4, 1, false)
	assert.Error(t, err)
}

func dateOrderPtr(o models.SortOrder) *models.SortOrder {
	return &o
}

// makeDailyPickUsecasePrices n本のうち最終日だけ大きく動く日足系列を生成する（domain_service側のテストヘルパーと同趣旨）。
func makeDailyPickUsecasePrices(n int, flatClose, lastClose decimal.Decimal) []*models.StockBrandDailyPrice {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]*models.StockBrandDailyPrice, n)
	for i := 0; i < n-1; i++ {
		out[i] = &models.StockBrandDailyPrice{
			Date:     base.AddDate(0, 0, i),
			Close:    flatClose,
			High:     flatClose.Add(decimal.NewFromInt(1)),
			Low:      flatClose.Sub(decimal.NewFromInt(1)),
			Volume:   2_000_000,
			Adjclose: flatClose,
		}
	}
	out[n-1] = &models.StockBrandDailyPrice{
		Date:     base.AddDate(0, 0, n-1),
		Close:    lastClose,
		High:     lastClose.Add(decimal.NewFromInt(5)),
		Low:      lastClose.Sub(decimal.NewFromInt(5)),
		Volume:   6_000_000,
		Adjclose: lastClose,
	}
	return out
}

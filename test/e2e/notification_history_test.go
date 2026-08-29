package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	"github.com/Code0716/stock-price-repository/entrypoint/api/router"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

// TestE2E_RecordingSlackAPIClient_RecordsNotificationHistory
// デコレータが実DBに記録することと、Slack送信を行う通知/行わない通知の振り分けを確認する。
func TestE2E_RecordingSlackAPIClient_RecordsNotificationHistory(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()
	helper.TruncateAllTables(t, db)

	repo := database.NewNotificationHistoryRepositoryImpl(db)
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("株情報系チャンネルはSlackへ送らず記録される", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		// 株情報系はSlackへ送信しないため rawClient に EXPECT を張らない
		rawClient := mock_gateway.NewMockSlackAPIClientRaw(ctrl)

		client := gateway.NewRecordingSlackAPIClient(rawClient, repo, zap.NewNop())
		body := "7203,トヨタ自動車,,現物買,100,2747.5,,2026-08-20"
		_, err := client.SendMessageByStrings(ctx, gateway.SlackChannelNameExchangeStockInfo, "三角持ち合い銘柄", &body, nil)
		require.NoError(t, err)

		notifications, err := repo.FindWithFilter(ctx, &models.NotificationHistoryFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Len(t, notifications, 1)
		assert.Equal(t, "三角持ち合い銘柄", notifications[0].Title)
		assert.Equal(t, &body, notifications[0].Body)
		assert.Equal(t, "株の情報交換", notifications[0].ChannelLabel)
		assert.Equal(t, models.NotificationHistorySourceSpr, notifications[0].Source)
	})

	t.Run("dev_notificationは記録されない", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		rawClient := mock_gateway.NewMockSlackAPIClientRaw(ctrl)
		rawClient.EXPECT().
			SendErrMessageNotification(gomock.Any(), gomock.Any()).
			Return(nil)

		client := gateway.NewRecordingSlackAPIClient(rawClient, repo, zap.NewNop())
		err := client.SendErrMessageNotification(ctx, assertErr{"boom"})
		require.NoError(t, err)

		count, err := repo.CountWithFilter(ctx, &models.NotificationHistoryFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

type assertErr struct{ msg string }

func (e assertErr) Error() string { return e.msg }

// TestE2E_GetNotificationHistories GET /notifications がフィルタ・ページネーションで正しく応答することを確認する。
func TestE2E_GetNotificationHistories(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()
	helper.TruncateAllTables(t, db)

	repo := database.NewNotificationHistoryRepositoryImpl(db)
	interactor := usecase.NewNotificationHistoryInteractor(repo)
	httpServer := driver.NewHTTPServer()
	notificationHandler := handler.NewNotificationHandler(interactor, httpServer, zap.NewNop())
	mux := router.NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, notificationHandler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx := context.Background()
	now := time.Now()

	seedBody := "本文"
	require.NoError(t, repo.Create(ctx, models.NewNotificationHistory("", models.NotificationHistorySourceStt, gateway.SlackChannelNameExchangeStockInfo.String(), "株の情報交換", "三角持ち合い銘柄", &seedBody, now)))
	require.NoError(t, repo.Create(ctx, models.NewNotificationHistory("", models.NotificationHistorySourceStt, gateway.SlackChannelNameMachineLearningResult.String(), "機械学習結果", "MLランキング", nil, now.Add(time.Second))))

	t.Run("channelで絞り込める", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/notifications?channel=" + gateway.SlackChannelNameExchangeStockInfo.String())
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result handler.GetNotificationHistoriesResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Len(t, result.Notifications, 1)
		assert.Equal(t, "三角持ち合い銘柄", result.Notifications[0].Title)
		assert.Equal(t, int64(1), result.Pagination.Total)
	})

	t.Run("絞り込みなしなら新しい順に全件返す", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/notifications")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result handler.GetNotificationHistoriesResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Len(t, result.Notifications, 2)
		assert.Equal(t, "MLランキング", result.Notifications[0].Title)
		assert.Equal(t, "三角持ち合い銘柄", result.Notifications[1].Title)
	})

	t.Run("qで本文・タイトルを検索できる", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/notifications?q=MLランキング")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result handler.GetNotificationHistoriesResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Len(t, result.Notifications, 1)
		assert.Equal(t, "MLランキング", result.Notifications[0].Title)
	})
}

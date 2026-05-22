package e2e

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
	"github.com/Code0716/stock-price-repository/entrypoint/api/router"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestE2E_Daytrade(t *testing.T) {
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	tx := database.NewTransaction(db)
	repo := database.NewDaytradeExecutionRepositoryImpl(db)
	interactor := usecase.NewDaytradeInteractor(tx, repo)

	httpServer := driver.NewHTTPServer()
	daytradeHandler := handler.NewDaytradeHandler(interactor, httpServer, zap.NewNop())
	mux := router.NewRouter(nil, nil, nil, nil, nil, nil, daytradeHandler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("初回インポートで3件挿入される", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		resp := postCSV(t, ts.URL, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		resp.Body.Close()

		assert.Equal(t, 3, result.TotalRow)
		assert.Equal(t, 3, result.Inserted)
		assert.Equal(t, 0, result.Skipped)
	})

	t.Run("同じCSVを再インポートすると重複スキップ", func(t *testing.T) {
		resp := postCSV(t, ts.URL, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		resp.Body.Close()

		assert.Equal(t, 3, result.TotalRow)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 3, result.Skipped)
	})

	t.Run("日次サマリーが正しく返る", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/summary?granularity=daily&from=2026-05-19&to=2026-05-21")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Granularity string                         `json:"granularity"`
			Buckets     []*models.DaytradeSummaryBucket `json:"buckets"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "daily", body.Granularity)
		assert.NotEmpty(t, body.Buckets)
	})

	t.Run("全期間サマリー (all) が1バケット返る", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/summary?granularity=all")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Buckets []*models.DaytradeSummaryBucket `json:"buckets"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Len(t, body.Buckets, 1)
		assert.Nil(t, body.Buckets[0].BucketDate)
		assert.Equal(t, 3, body.Buckets[0].TradeCount)
	})

	t.Run("指定日の明細取得", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/executions?date=2026-05-21")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Executions []*models.DaytradeExecution `json:"executions"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotEmpty(t, body.Executions)
	})

	t.Run("カバー期間の取得", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/range")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Min *string `json:"min"`
			Max *string `json:"max"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotNil(t, body.Min)
		assert.NotNil(t, body.Max)
	})

	t.Run("granularityが不正で400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/summary?granularity=invalid")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("dateなしで400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/executions")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("fileフィールドなしで400", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/daytrade/executions/import", "multipart/form-data; boundary=xxx", bytes.NewReader([]byte("--xxx--")))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func postCSV(t *testing.T, baseURL string, _ []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "test.csv")
	require.NoError(t, err)

	csvBytes, err := os.ReadFile("../../usecase/daytrade/testdata/sbi_sample_sjis.csv")
	require.NoError(t, err)
	_, err = fw.Write(csvBytes)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	resp, err := http.Post(baseURL+"/daytrade/executions/import", mw.FormDataContentType(), &buf)
	require.NoError(t, err)
	return resp
}

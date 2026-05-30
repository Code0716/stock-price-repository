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

		resp := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_sjis.csv")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		resp.Body.Close()

		assert.Equal(t, 3, result.TotalRow)
		assert.Equal(t, 3, result.Inserted)
		assert.Equal(t, 0, result.Skipped)
		assert.Equal(t, 0, result.Deleted)
	})

	t.Run("同じCSVを再インポートするとその日が置き換わる", func(t *testing.T) {
		// 前のテストで 3 件挿入済み（TruncateAllTables なし）
		resp := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_sjis.csv")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		resp.Body.Close()

		assert.Equal(t, 3, result.TotalRow)
		assert.Equal(t, 3, result.Inserted)
		assert.Equal(t, 0, result.Skipped)
		assert.Equal(t, 3, result.Deleted) // 同じ日付のレコードを削除してから再挿入
	})

	t.Run("日次サマリーが正しく返る", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/summary?granularity=daily&from=2026-05-19&to=2026-05-21")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Granularity string                          `json:"granularity"`
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
		assert.Equal(t, int64(3500), body.Buckets[0].GrossProfit)
		assert.Equal(t, int64(-800), body.Buckets[0].GrossLoss)
		assert.Equal(t, 2, body.Buckets[0].WinCount)
		assert.Equal(t, 1, body.Buckets[0].LossCount)
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

	// --- /daytrade/stats エンドポイントのテスト ---

	t.Run("period stats_全期間で正しい集計が返る", func(t *testing.T) {
		// 事前条件: sbi_sample_sjis.csv の 3 件が挿入済み
		// profitLoss: +1460(9984), +2040(9984), -800(5803) = 合計 2700
		// maxProfit=2040, maxLoss=-800, winCount=2, lossCount=1
		resp, err := http.Get(ts.URL + "/daytrade/stats")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body models.DaytradePeriodStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		assert.Equal(t, int64(2700), body.ProfitLoss)
		assert.Equal(t, 3, body.TradeCount)
		assert.Equal(t, int64(3500), body.GrossProfit)
		assert.Equal(t, int64(-800), body.GrossLoss)
		assert.Equal(t, 2, body.WinCount)
		assert.Equal(t, 1, body.LossCount)
		assert.Equal(t, int64(2040), body.MaxProfit)
		assert.Equal(t, int64(-800), body.MaxLoss)
		assert.GreaterOrEqual(t, body.MaxDrawdown, int64(0))
		assert.GreaterOrEqual(t, body.MaxRunup, int64(0))
		assert.GreaterOrEqual(t, body.MaxLossStreak, 0)
	})

	t.Run("period stats_from/toで絞り込み", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/stats?from=2026-05-21&to=2026-05-21")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body models.DaytradePeriodStats
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		assert.Equal(t, int64(660), body.ProfitLoss)   // 1460 + (-800)
		assert.Equal(t, 2, body.TradeCount)
	})

	t.Run("period stats_from>toで400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/stats?from=2026-05-31&to=2026-05-01")
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

	// --- 銘柄別集計エンドポイントのテスト ---

	t.Run("銘柄別集計_全期間で2銘柄返りprofitLoss降順", func(t *testing.T) {
		// 事前条件: CSV 3件（9984×2, 5803×1）が挿入済み
		resp, err := http.Get(ts.URL + "/daytrade/summary-by-symbol")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			From  *string                         `json:"from"`
			To    *string                         `json:"to"`
			Items []*models.DaytradeSymbolSummary `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Nil(t, body.From)
		assert.Nil(t, body.To)
		require.Len(t, body.Items, 2)

		// profit_loss DESC: 9984 (3500) が先、5803 (-800) が後
		assert.Equal(t, "9984", body.Items[0].TickerSymbol)
		assert.Equal(t, int64(3500), body.Items[0].ProfitLoss) // 1460 + 2040
		assert.Equal(t, 2, body.Items[0].TradeCount)
		assert.Equal(t, int64(3500), body.Items[0].GrossProfit)
		assert.Equal(t, int64(0), body.Items[0].GrossLoss)
		assert.Equal(t, 2, body.Items[0].WinCount)
		assert.Equal(t, 0, body.Items[0].LossCount)

		assert.Equal(t, "5803", body.Items[1].TickerSymbol)
		assert.Equal(t, int64(-800), body.Items[1].ProfitLoss)
		assert.Equal(t, int64(0), body.Items[1].GrossProfit)
		assert.Equal(t, int64(-800), body.Items[1].GrossLoss)
		assert.Equal(t, 1, body.Items[1].TradeCount)
		assert.Equal(t, 0, body.Items[1].WinCount)
		assert.Equal(t, 1, body.Items[1].LossCount)
	})

	t.Run("銘柄別集計_from/toで絞り込み", func(t *testing.T) {
		// 2026-05-21 のみ: 9984(+1460) と 5803(-800) が対象
		resp, err := http.Get(ts.URL + "/daytrade/summary-by-symbol?from=2026-05-21&to=2026-05-21")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			From  *string                         `json:"from"`
			To    *string                         `json:"to"`
			Items []*models.DaytradeSymbolSummary `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.NotNil(t, body.From)
		require.NotNil(t, body.To)
		assert.Equal(t, "2026-05-21", *body.From)
		assert.Equal(t, "2026-05-21", *body.To)
		require.Len(t, body.Items, 2)

		// 2026-05-21 の 9984 は 1 件 (+1460)
		assert.Equal(t, "9984", body.Items[0].TickerSymbol)
		assert.Equal(t, int64(1460), body.Items[0].ProfitLoss)
		assert.Equal(t, 1, body.Items[0].TradeCount)
		assert.Equal(t, int64(1460), body.Items[0].GrossProfit)
		assert.Equal(t, int64(0), body.Items[0].GrossLoss)
	})

	t.Run("銘柄別集計_from>toで400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/daytrade/summary-by-symbol?from=2026-05-31&to=2026-05-01")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("銘柄別集計_stock_brand登録済みは最新名、未登録はCSV名にフォールバック", func(t *testing.T) {
		// stock_brand に 9984 を登録（有効なレコード）
		require.NoError(t, db.Exec(
			"INSERT INTO stock_brand (id, ticker_symbol, name, market_code, market_name, created_at, updated_at)"+
				" VALUES (UUID(), '9984', 'ソフトバンクグループ株式会社', '111', '東証プライム', NOW(), NOW())",
		).Error)
		defer func() {
			db.Exec("DELETE FROM stock_brand WHERE ticker_symbol = '9984'")
		}()

		resp, err := http.Get(ts.URL + "/daytrade/summary-by-symbol")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Items []*models.DaytradeSymbolSummary `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body.Items, 2)

		// 9984: stock_brand の name を使う
		assert.Equal(t, "9984", body.Items[0].TickerSymbol)
		assert.Equal(t, "ソフトバンクグループ株式会社", body.Items[0].BrandName)

		// 5803: stock_brand 未登録なので CSV の brand_name
		assert.Equal(t, "5803", body.Items[1].TickerSymbol)
		assert.NotEmpty(t, body.Items[1].BrandName)
	})

	t.Run("銘柄別集計_stock_brand論理削除済みはCSV名にフォールバック", func(t *testing.T) {
		// 論理削除済みの stock_brand を登録
		require.NoError(t, db.Exec(
			"INSERT INTO stock_brand (id, ticker_symbol, name, market_code, market_name, created_at, updated_at, deleted_at)"+
				" VALUES (UUID(), '9984', '削除済み銘柄名', '111', '東証プライム', NOW(), NOW(), NOW())",
		).Error)
		defer func() {
			db.Exec("DELETE FROM stock_brand WHERE ticker_symbol = '9984'")
		}()

		resp, err := http.Get(ts.URL + "/daytrade/summary-by-symbol")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Items []*models.DaytradeSymbolSummary `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Len(t, body.Items, 2)

		// 論理削除済みなので b.name は NULL → d.brand_name にフォールバック
		assert.Equal(t, "9984", body.Items[0].TickerSymbol)
		assert.NotEqual(t, "削除済み銘柄名", body.Items[0].BrandName) // 削除済み名は使わない
		assert.NotEmpty(t, body.Items[0].BrandName)            // CSV 由来の名前
	})

	// --- 日付範囲で置き換えのテスト ---

	t.Run("日付範囲で置き換え_同じ日のCSVを別フォーマットで再取り込みすると新内容に置き換わる", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		// 旧フォーマット CSV を 3 件挿入（trade_kind="売建" あり）
		resp := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_sjis.csv")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 新フォーマット CSV（同じ日付・同数）で再取り込み
		resp2 := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_new_sjis.csv")
		require.Equal(t, http.StatusOK, resp2.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))
		resp2.Body.Close()

		assert.Equal(t, 3, result.Deleted)  // 旧フォーマット 3 件が日付範囲で削除された
		assert.Equal(t, 3, result.Inserted) // 新フォーマット 3 件が挿入された
		assert.Equal(t, 0, result.Skipped)
		assert.Equal(t, 3, result.TotalRow)

		// DB の内容が新フォーマットになっていること（trade_kind が空文字）
		resp3, err := http.Get(ts.URL + "/daytrade/executions?date=2026-05-21")
		require.NoError(t, err)
		defer resp3.Body.Close()

		var body struct {
			Executions []*models.DaytradeExecution `json:"executions"`
		}
		require.NoError(t, json.NewDecoder(resp3.Body).Decode(&body))
		require.NotEmpty(t, body.Executions)
		for _, ex := range body.Executions {
			assert.Equal(t, "", ex.TradeKind, "新フォーマットはtradeKind空文字")
		}
	})

	t.Run("日付範囲で置き換え_別日のCSVを取り込んでも既存日付は消えない", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		// 2026-05-19 と 2026-05-21 の 3 件を挿入
		resp := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_sjis.csv")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 別の日（2026-05-14）だけを含む inline CSV を取り込む
		header := `"約定日","取引","銘柄","信用","数量","約定代金","単価","平均取得単価","売買損益(税引前・円)"` + "\n"
		row := `"2026/5/14","売建","ソフトバンクグループ 9984","返済売","100","596,940","5,969.4","5,960","+840"` + "\n"
		csvContent := []byte(header + row)

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("file", "new_day.csv")
		require.NoError(t, err)
		_, err = fw.Write(csvContent)
		require.NoError(t, err)
		require.NoError(t, mw.Close())

		resp2, err := http.Post(ts.URL+"/daytrade/executions/import", mw.FormDataContentType(), &buf)
		require.NoError(t, err)
		defer resp2.Body.Close()
		require.Equal(t, http.StatusOK, resp2.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))

		assert.Equal(t, 1, result.TotalRow)
		assert.Equal(t, 1, result.Inserted)
		assert.Equal(t, 0, result.Deleted) // 2026-05-14 には既存レコードなし

		// 元の 3 件（2026-05-19, 2026-05-21）が残っていること
		var allStats struct {
			Buckets []*models.DaytradeSummaryBucket `json:"buckets"`
		}
		resp3, err := http.Get(ts.URL + "/daytrade/summary?granularity=all")
		require.NoError(t, err)
		defer resp3.Body.Close()
		require.NoError(t, json.NewDecoder(resp3.Body).Decode(&allStats))
		assert.Equal(t, 4, allStats.Buckets[0].TradeCount) // 元 3 件 + 新規 1 件
	})

	t.Run("新フォーマットCSV_tradeKindが空文字でmarginKindが正しい", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		resp := postCSV(t, ts.URL, "../../usecase/daytrade/testdata/sbi_sample_new_sjis.csv")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 新フォーマットで取り込まれたレコードの trade_kind / margin_kind を確認
		resp2, err := http.Get(ts.URL + "/daytrade/executions?date=2026-05-21")
		require.NoError(t, err)
		defer resp2.Body.Close()

		var body struct {
			Executions []*models.DaytradeExecution `json:"executions"`
		}
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
		require.NotEmpty(t, body.Executions)

		for _, ex := range body.Executions {
			assert.Equal(t, "", ex.TradeKind, "新フォーマットはtradeKind空文字")
			assert.Contains(t, []string{"返済売", "返済買"}, ex.MarginKind)
		}
	})

	t.Run("occurrence_no_重複約定が全件挿入される", func(t *testing.T) {
		helper.TruncateAllTables(t, db)

		// 同一自然キーの行を 2 行含む inline CSV を POST
		header := `"約定日","取引","銘柄","信用","数量","約定代金","単価","平均取得単価","売買損益(税引前・円)"` + "\n"
		row := `"2026/5/14","売建","ソフトバンクグループ 9984","返済売","100","596,940","5,969.4","5,960","+840"` + "\n"
		csvContent := []byte(header + row + row) // 2 行同一キー

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("file", "dup.csv")
		require.NoError(t, err)
		_, err = fw.Write(csvContent)
		require.NoError(t, err)
		require.NoError(t, mw.Close())

		resp, err := http.Post(ts.URL+"/daytrade/executions/import", mw.FormDataContentType(), &buf)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.DaytradeImportResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, 2, result.TotalRow)
		assert.Equal(t, 2, result.Inserted) // occurrence_no で区別されるため 2 件とも挿入
		assert.Equal(t, 0, result.Skipped)
	})
}

func postCSV(t *testing.T, baseURL string, csvPath string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "test.csv")
	require.NoError(t, err)

	csvBytes, err := os.ReadFile(csvPath)
	require.NoError(t, err)
	_, err = fw.Write(csvBytes)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	resp, err := http.Post(baseURL+"/daytrade/executions/import", mw.FormDataContentType(), &buf)
	require.NoError(t, err)
	return resp
}

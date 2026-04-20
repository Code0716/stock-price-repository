# gateway/ — 外部 API クライアント

このパッケージはすべての外部 API 通信を担う。ビジネスロジックは持たない。

## インターフェース

`StockAPIClient` (`stock_api_client.go`) が全メソッドを定義。j-Quants と Yahoo Finance の両 API を束ねている。新規メソッド追加時はここに先に定義してから実装を書く。

## j-Quants API の構成

### 認証フロー

1. `/token/auth_user` にメールアドレス・パスワードを POST → **リフレッシュトークン**を取得
2. `/token/auth_refresh` にリフレッシュトークンを POST → **ID トークン**を取得
3. 以降の全リクエストの `Authorization: Bearer <IDトークン>` ヘッダに ID トークンをセット

リフレッシュトークンは Redis に保存（TTL 管理あり）。ID トークンは毎回リフレッシュから取得する。

### 主要エンドポイント

| メソッド | j-Quants エンドポイント | 用途 |
|----------|------------------------|------|
| `GetStockBrands` | `/listed/info` | 上場銘柄一覧 |
| `GetAllBrandDailyPricesByDate` | `/prices/daily_quotes` | 指定日の全銘柄日足 |
| `GetDailyPricesBySymbolAndRange` | `/prices/daily_quotes` | 期間指定の銘柄日足 |
| `GetFinancialStatementsBySymbol` | `/fins/statements` | 銘柄の財務諸表 |
| `GetFinancialStatementsByDate` | `/fins/statements` | 日付の財務諸表 |
| `GetAnnounceFinSchedule` | `/fins/announcement` | 決算発表スケジュール |
| `GetTradingCalendarsInfo` | `/markets/trading_calendar` | 取引カレンダー |

### レスポンス型の配置

- レスポンス全体のラッパー構造体: `stock_api_response_info_models.go`
- 銘柄・価格などのドメイン近いモデル: `stock_api_model.go`
- j-Quants 固有の定数・型: `j_quants_api.go`

## 新規 j-Quants エンドポイント追加手順

1. **インターフェースにメソッドを追加** (`stock_api_client.go`)
2. **レスポンス型を定義** (`stock_api_response_info_models.go` または新規ファイル)
3. **実装を追加**（既存の `*StockAPIClientImpl` メソッドに追記）
4. **`make mock`** でモックを再生成
5. **`make di`** で Wire を再生成
6. 対応する usecase と CLI コマンドを追加

## エラーハンドリング

- HTTP エラー（4xx/5xx）は `errors.Errorf("j-Quants API error: status=%d", resp.StatusCode)` で返す
- レスポンス JSON のデコードエラーは `errors.Wrap(err, "json decode error")` でラップ
- API 側のエラーレスポンス（`{"message": "..."}` 形式）は message を含めてラップ

## レート制限

j-Quants API には呼び出し頻度制限がある。大量銘柄をループ処理する場合は `time.Sleep` や `errgroup` で制御すること（既存の `newStockBrandDailyPrices` を参考に）。

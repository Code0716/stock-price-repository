# /add-jquants-endpoint

j-Quants API の新規エンドポイントを追加し、対応する CLI コマンドまで一通り実装する。

## 引数

追加したいエンドポイントの概要（例: "オプション情報取得 /option/index_option"）を教えてください。

## 実装手順

以下の順序で実装してください:

### 1. レスポンス型の定義
`infrastructure/gateway/stock_api_response_info_models.go` に j-Quants API のレスポンス構造体を追加する。

### 2. インターフェースにメソッドを追加
`infrastructure/gateway/stock_api_client.go` の `StockAPIClient` インターフェースに新規メソッドを追加する。

### 3. 実装を追加
既存の `*StockAPIClientImpl` に実装メソッドを追加する。
- 認証ヘッダ: `Authorization: Bearer <IDトークン>` （既存メソッドのパターンを踏襲）
- エラーは `errors.Wrap` でラップ
- レスポンスを `stock_api_response_info_models.go` の型にデコードしてから `stock_api_model.go` のドメイン近い型に変換して返す

### 4. モデルが必要な場合
- `models/` にドメインモデルを追加
- DB 保存が必要なら `sql/migrations/` に DDL を追加 → `make migrate-up` → `make gen`

### 5. リポジトリが必要な場合
- `repositories/` にインターフェースを追加
- `infrastructure/database/` に GORM 実装を追加（`GetTxQuery` パターン必須）

### 6. usecase を追加
- 適切なインタラクターファイル（`usecase/*_interactor.go`）にメソッドを追加
- 書き込み処理は `tx.DoInTx` でラップ

### 7. CLI コマンドを追加
- `entrypoint/cli/commands/` に新規コマンドファイルを追加
- `di/wire.go` の該当 Set に登録

### 8. コード生成
```bash
make mock   # モック再生成
make di     # Wire 再生成
```

### 9. テストを書く
- `infrastructure/gateway/*_test.go`: API クライアントのユニットテスト
- `usecase/*_test.go`: usecase のユニットテスト（モック使用）
- `test/e2e/`: E2E テスト（`helper.SetupTestDB` + 外部 API モック）

## 注意事項
- j-Quants の有料プランでしか使えないエンドポイントはコメントに明記する
- ページネーションがある場合は全件取得のループ処理を実装する（既存の `GetAllBrandDailyPricesByDate` を参考に）

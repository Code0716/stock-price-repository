# /add-usecase

新規 usecase を追加する。

## 引数

追加したい usecase の概要（例: "銘柄の配当情報を取得して保存する"）を教えてください。

## 実装手順

### 1. インタラクターを特定または作成

既存インタラクターに追加できる場合はそこに追記する:
- `usecase/stock_brands_interactor.go` — 銘柄マスタ系
- `usecase/stock_brands_daily_stock_price_interactor.go` — 日足データ系
- `usecase/index_interactor.go` — 指数系

新機能の場合は新規 `usecase/<機能名>_interactor.go` を作成し、インターフェースと実装構造体を定義する。

### 2. usecase メソッドを実装

- 読み取り専用: repository の `Find` 系メソッドを呼ぶ
- 書き込みあり: 必ず `tx.DoInTx` でラップ
- 外部 API 取得: `stockAPIClient` 経由（直接 HTTP 呼び出し禁止）
- エラーは全て `errors.Wrap` でラップ

### 3. テストを書く

`usecase/<機能名>_test.go` をテーブル駆動テストで作成:
- 正常系・異常系・境界値を網羅
- 依存インターフェースは全て `gomock` でモック
- `gomock.Any()` は避け具体的な引数で検証

### 4. DI 登録

`di/wire.go` の該当 `var xxxSet` にコンストラクタを追加 → `make di`

### 5. CLI コマンドから呼ぶ

`entrypoint/cli/commands/` に呼び出し元コマンドを追加（既存の他コマンドを参考に）。

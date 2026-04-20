# /add-repository

新規リポジトリ（インターフェース + DB 実装）を追加する。

## 引数

追加したいリポジトリの概要（例: "配当情報テーブルの CRUD"）を教えてください。

## 実装手順

### 1. マイグレーションファイルを作成（テーブルが新規の場合）

```bash
make migrate-file name=<テーブル名>
```

`.up.sql` に CREATE TABLE、`.down.sql` に DROP TABLE を記述 → `make migrate-up` → `make gen`

### 2. ドメインモデルを追加

`models/<モデル名>.go` にドメインエンティティを定義。`gen_model/` の構造体とは別に作成する。

### 3. インターフェースを定義

`repositories/<機能名>.go` に `//go:generate` ディレクティブとインターフェースを定義:

```go
//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

type MyRepository interface {
    FindAll(ctx context.Context) ([]*models.MyModel, error)
    Upsert(ctx context.Context, items []*models.MyModel) error
}
```

### 4. DB 実装を追加

`infrastructure/database/<機能名>.go` に実装を作成:
- `GetTxQuery(ctx)` パターン必須
- `convertToDomainModel` / `convertToDBModel` で変換
- `errors.Wrap` でエラーをラップ

### 5. コード生成

```bash
make mock   # モック再生成
make di     # Wire 再生成（di/wire.go に登録後）
```

### 6. DI 登録

`di/wire.go` の該当 `var xxxSet` に新コンストラクタを追加 → `make di`

### 7. テストを書く

- `infrastructure/database/<機能名>_test.go`: DB 実装のユニットテスト
- `repositories/` のインターフェースに対する E2E テストを `test/e2e/` に追加

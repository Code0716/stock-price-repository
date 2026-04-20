# infrastructure/database/ — DB アクセス層

## 構成

```
database/
  *.go          repositories インターフェースの GORM 実装
  transaction.go  GetTxQuery パターンの実装（必ず参照）
  gen_model/    GORM Gen 自動生成 DB 構造体（編集禁止）
  gen_query/    GORM Gen 自動生成クエリ層（編集禁止）
  setup_test.go E2E テスト用 DB セットアップ
```

## GetTxQuery パターン（必須）

書き込みを行う全メソッドで使用する。`transaction.go` を参照:

```go
func (r *MyRepositoryImpl) Upsert(ctx context.Context, items []*models.MyModel) error {
    tx, ok := GetTxQuery(ctx)
    if !ok {
        tx = r.query  // トランザクション外の場合は通常クエリを使用
    }
    // tx.MyTable.xxx() で操作
}
```

読み取り専用メソッドでも GetTxQuery パターンを統一して使うこと（トランザクション内で読み取る場合があるため）。

## GORM Gen の使い方

- `gen_query/` のメソッドで型安全なクエリを構築する
- 生 SQL は書かない
- `gen_model/` の構造体は DB との変換にのみ使用し、usecase 層に渡す前に `models/` に変換する

```go
// 変換パターン例
func (r *StockBrandRepositoryImpl) convertToDomainModel(m *gen_model.StockBrand) *models.StockBrand {
    return &models.StockBrand{
        ID:           m.ID,
        TickerSymbol: m.TickerSymbol,
        // ...
    }
}
```

## 新規テーブル追加手順

1. `make migrate-file name=<テーブル名>` でマイグレーションファイルを作成
2. `.up.sql` に CREATE TABLE、`.down.sql` に DROP TABLE を記述
3. `make migrate-up` を実行
4. **`make gen`** で `gen_model/` と `gen_query/` を再生成（DB 接続必要）
5. `models/` に対応するドメインモデルを追加
6. `repositories/` にインターフェースを追加
7. このパッケージに実装ファイルを追加（`GetTxQuery` + `convertToDomainModel/convertToDBModel`）
8. `make mock && make di`

## 禁止事項

- `gen_model/` と `gen_query/` を手動編集しない
- `gorm.DB` を直接クエリに使用しない（`gen_query/` を経由する）
- ビジネスロジックをこの層に書かない（変換処理のみ）

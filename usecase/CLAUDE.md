# usecase/ — ビジネスロジック層

## 責務

- ビジネスロジックの**オーケストレーション**のみ
- トランザクション境界の管理（`tx.DoInTx` を呼ぶのはこの層だけ）
- ドメインモデル（`models/`）の操作
- 外部 API 呼び出しは必ず `gateway.StockAPIClient` 経由

## 禁止事項

- `infrastructure/` や `gen_model/` を直接インポートしない
- DB クエリや HTTP リクエストをこの層に書かない
- トランザクションを repository 内で開始しない

## インタラクター構成

各機能はインタラクター構造体にまとめている:

| ファイル | インタラクター | 主な用途 |
|----------|---------------|---------|
| `stock_brands_interactor.go` | `stockBrandsInteractorImpl` | 銘柄マスタ管理 |
| `stock_brands_daily_stock_price_interactor.go` | `stockBrandsDailyStockPriceInteractorImpl` | 日足データ管理 |
| `index_interactor.go` | `indexInteractorImpl` | 日経・DJI 指数管理 |

## 新規 usecase 追加手順

1. **インタラクターが既存なら既存ファイルに追記**。新機能なら新規 `*_interactor.go` を作成
2. **メソッドシグネチャを定義**（`entrypoint/cli/commands/` から呼ばれる形を先に決める）
3. **実装を書く**
   - DB アクセスは repository インターフェース経由
   - 外部 API は `StockAPIClient` 経由
   - 書き込みは `tx.DoInTx` でラップ
4. **テストを書く**（テーブル駆動、モック使用）
5. **DI 登録**: `di/wire.go` の該当 Set に追加 → `make di`

## トランザクションパターン

```go
func (si *myInteractorImpl) DoSomething(ctx context.Context) error {
    err := si.tx.DoInTx(ctx, func(ctx context.Context) error {
        if err := si.someRepository.Write(ctx, ...); err != nil {
            return errors.Wrap(err, "someRepository.Write error")
        }
        return nil
    })
    if err != nil {
        return errors.Wrap(err, "DoInTx error")
    }
    return nil
}
```

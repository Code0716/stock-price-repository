# test/ — テスト方針

## ディレクトリ構成

```
test/
  e2e/      E2E テスト（実 DB 使用）
  helper/   共通ヘルパー（SetupTestDB, TruncateAllTables 等）
```

ユニットテストは各実装ファイルと同ディレクトリの `*_test.go` に配置する。

## E2E テストの基本構成

```go
func TestE2E_CommandName(t *testing.T) {
    db, cleanup := helper.SetupTestDB(t)  // テスト専用 DB を作成・マイグレーション適用
    defer cleanup()

    // miniredis でRedisをモック
    // mock/gateway で外部 API をモック

    tests := []struct {
        name    string
        args    struct{ cmdArgs []string }
        setup   func(t *testing.T)   // 初期データ投入
        wantErr bool
        check   func(t *testing.T)   // DB状態の検証
    }{...}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            helper.TruncateAllTables(t, db)  // テストケース間でクリーンアップ
            if tt.setup != nil { tt.setup(t) }

            err := runner.Run(context.Background(), tt.args.cmdArgs)
            if (err != nil) != tt.wantErr {
                t.Errorf(...)
            }
            if tt.check != nil { tt.check(t) }
        })
    }
}
```

## ルール

- **外部 API は必ずモック**: `mock/gateway` の `MockStockAPIClient` を使用。j-Quants / Yahoo Finance の実 API は絶対に呼ばない
- **Redis は miniredis**: `github.com/alicebob/miniredis` でインメモリ Redis をモック
- **`ENVCODE=e2e`** 環境変数が必要（自動設定済みの場合は不要）
- **DB は `helper.SetupTestDB(t)`**: ランダム名の専用 DB を作成し全マイグレーションを自動適用
- `helper.TruncateAllTables` は各テストケース実行前に呼ぶ（`defer` ではなくループ先頭で）

## ユニットテストのルール

- テーブル駆動テスト形式（`tests := []struct{...}{...}` + `for _, tt := range tests`）
- プライベート関数も含め全関数にテストを書く
- `gomock.Any()` は避け、可能な限り具体的な引数で検証
- 使用しないモックフィールドは `nil` にして初期化関数を省略
- アサーションは `testify/assert` を使用

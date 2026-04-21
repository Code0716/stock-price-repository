# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

Go 1.26 製の株価データ収集システム。**j-Quants API** を主軸に Yahoo Finance API からデータを取得し MySQL に保存する。REST API / gRPC API / CLI の3つの実行形態を持つが、**コアはデータ収集 CLI**。Raspberry Pi 上で常時稼働する想定。モジュールパス: `github.com/Code0716/stock-price-repository`。

**応答言語**: コード説明・レビュー・提案はすべて日本語で行う。

---

## 必須コマンド

```bash
# セットアップ
make install-tools          # wire, goimports, golangci-lint, air, govulncheck をインストール

# コード生成 — モデル・インターフェース変更後は必ずこの順序で実行
make gen                    # GORM Gen → gen_model/ と gen_query/ を再生成（DB接続必要）
make mock                   # mockgen → mock/ を全削除して再生成
make di                     # wire gen → di/wire_gen.go を再生成

# ローカル起動
make cli command=<name>     # CLIコマンド実行 (例: make cli command=health_check)
make api                    # HTTP API サーバー (air ホットリロード, port 8080)
make grpc-server            # gRPC サーバー (air ホットリロード, port 50051)
docker compose up           # フルスタック (MySQL + Redis + api + grpc-server)

# DB マイグレーション
make migrate-file name=<名前>  # 新規マイグレーションファイル作成
make migrate-up                # 未適用マイグレーションを全適用
make migrate-down              # 直近1件をロールバック

# テスト・品質管理
make test                   # unit + e2e + 脆弱性スキャン（CI相当）
make test-unit              # ユニットテストのみ (ENVCODE=unit)
make test-e2e               # E2Eテストのみ (ENVCODE=e2e, Docker MySQL 必要)
make lint                   # golangci-lint

# 個別テスト実行
ENVCODE=unit go test -v -run TestStockBrandInteractor_FindAll ./usecase/
ENVCODE=e2e  go test -v -run TestE2E_CommandName ./test/e2e/...

# Proto（別リポジトリ: Code0716/stock-price-proto）
make proto-setup            # 初回のみ proto-definitions/ をクローン
make proto-pull             # proto 定義を最新に更新
make proto-gen              # buf generate → pb/ を再生成
```

---

## アーキテクチャ構成

Clean Architecture。依存方向: `entrypoint → usecase → repositories (interface) ← infrastructure (実装)`

```
entrypoint/
  cli/            urfave/cli/v2 CLI、Wire でブートストラップ
  api/            HTTP ハンドラー + ルーター
  grpc/           gRPC サーバーエントリポイント + server/ 実装
usecase/          ビジネスロジック（オーケストレーション）
repositories/     インターフェース定義のみ（実装なし）
infrastructure/
  database/       repositories インターフェースの GORM 実装
    gen_model/    GORM Gen 自動生成 DB 構造体（編集禁止）
    gen_query/    GORM Gen 自動生成クエリ層（編集禁止）
  gateway/        外部 API クライアント（j-Quants, Yahoo Finance, Slack）
  cli/            CLI コマンド実装
models/           ドメインエンティティ（gen_model とは別）
driver/           DB / Redis / ロガー初期化
di/               Wire DI 設定（wire.go + 生成された wire_gen.go）
mock/             自動生成モック（ソースのパッケージ構造をミラー）
_gorm_gen/        GORM Gen ジェネレータスクリプト
sql/
  _init_sql/      初期スキーマ DDL
  migrations/     golang-migrate の .up.sql / .down.sql ペア
pb/               生成された protobuf Go コード（編集禁止）
test/
  e2e/            Docker MySQL を使った E2E テスト
  helper/         SetupTestDB, TruncateAllTables などの共通ヘルパー
```

---

## 重要な規約

### 1. エラーハンドリング — 必ず `github.com/pkg/errors`

```go
// 正しい
return errors.Wrap(err, "StockBrandRepositoryImpl.FindAll error")

// 禁止
return err                        // スタックトレースが失われる
return fmt.Errorf("...: %w", err) // パッケージ違い
_ = riskyCall()                   // エラー無視禁止
```

### 2. DB トランザクション — `GetTxQuery(ctx)` パターン

書き込みを行う全リポジトリメソッドでトランザクションを引き継ぐ:

```go
func (r *MyRepositoryImpl) Write(ctx context.Context, ...) error {
    tx, ok := GetTxQuery(ctx)
    if !ok {
        tx = r.query
    }
    // r.query ではなく tx を使用する
}
```

トランザクションの開始は **usecase 層のみ**（`tx.DoInTx`）。repository や infrastructure では開始しない。

### 3. モデル分離 — ドメイン vs DB

- `models/` = ドメインエンティティ（usecase 以上でのみ使用）
- `infrastructure/database/gen_model/` = GORM DB 構造体
- 変換は repository 実装内の `convertToDomainModel` / `convertToDBModel` で行う
- **usecase 層が `gen_model` をインポートしてはいけない**

### 4. 命名規則

| 対象 | 規則 | 例 |
|------|------|----|
| インターフェース | `-er` サフィックス | `StockBrandRepository`, `StockAPIClienter` |
| 実装 | `Impl` サフィックス | `StockBrandRepositoryImpl` |
| 未エクスポート | camelCase | `createDailyStockPrice` |
| エクスポート定数 | PascalCase | `JQuantsMarketCodePrime` |

### 5. `//go:generate` ディレクティブ

モック化可能なインターフェースを定義する全ファイルに記述:

```go
//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
```

`make mock` は `mock/` を全削除してから再生成する。

### 6. DI 登録

新規コンポーネント追加時:
1. `di/wire.go` の該当 `var xxxSet` にコンストラクタを追加
2. `make di` で `di/wire_gen.go` を再生成
3. `wire_gen.go` の手動編集は禁止

---

## コード生成ワークフロー

DB テーブル・ドメインモデル・インターフェースを変更したら **必ずこの順序**で実行:

```bash
make gen    # 1. GORM Gen（DB接続が必要）
make mock   # 2. mockgen
make di     # 3. Wire
```

順序を守らないとコンパイルエラーが発生する。

---

## テスト

### ユニットテスト構造

`fields` 構造体の各フィールドは `func(ctrl *gomock.Controller) Interface` 型とし、テストケースごとに必要なモックのみ初期化する。使用しないフィールドは `nil` で省略する:

```go
func TestService_Method(t *testing.T) {
    type fields struct {
        mainRepo func(ctrl *gomock.Controller) repository.MainRepository
        subRepo  func(ctrl *gomock.Controller) repository.SubRepository // 不要なケースは nil
    }
    tests := []struct {
        name    string
        fields  fields
        args    args
        want    *model.Result
        wantErr bool
    }{
        {
            name: "正常系",
            fields: fields{
                mainRepo: func(ctrl *gomock.Controller) repository.MainRepository {
                    m := mockrepository.NewMockMainRepository(ctrl)
                    m.EXPECT().Find(gomock.Any(), gomock.Eq(uint64(1))).Return(&model.Entity{}, nil)
                    return m
                },
                // subRepo はこのケースで不使用のため省略（nil）
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            s := &service{mainRepo: tt.fields.mainRepo(ctrl)}
            if tt.fields.subRepo != nil {
                s.subRepo = tt.fields.subRepo(ctrl)
            }

            got, err := s.Method(tt.args.ctx, tt.args.id)
            if (err != nil) != tt.wantErr {
                t.Errorf("Method() error = %v, wantErr %v", err, tt.wantErr)
            }
            assert.Equal(t, tt.want, got)
        })
    }
}
```

- プライベート関数を含む全関数に 1対1 でテスト作成
- `gomock.Any()` は避け、`gomock.Eq` / `DoAndReturn` で引数を厳密に検証

### E2E テスト構造

```go
func TestE2E_CommandName(t *testing.T) {
    db, cleanup := helper.SetupTestDB(t)  // ランダム名の専用 DB を作成・全マイグレーション適用
    defer cleanup()

    // Redis は miniredis でモック（github.com/alicebob/miniredis）
    // 外部 API は mock/gateway の MockStockAPIClient を使用

    tests := []struct {
        name    string
        setup   func(t *testing.T)
        wantErr bool
        check   func(t *testing.T)
    }{...}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            helper.TruncateAllTables(t, db)  // defer ではなくループ先頭で呼ぶ
            if tt.setup != nil { tt.setup(t) }

            err := runner.Run(context.Background(), tt.args.cmdArgs)
            // assert...
            if tt.check != nil { tt.check(t) }
        })
    }
}
```

---

## gateway/ — 外部 API クライアント

`StockAPIClient` (`infrastructure/gateway/stock_api_client.go`) が全メソッドを定義。新規メソッド追加時はインターフェースに先に定義してから実装を書く。

### j-Quants 認証フロー

1. `/token/auth_user` → **リフレッシュトークン**取得
2. `/token/auth_refresh` → **ID トークン**取得
3. 全リクエストの `Authorization: Bearer <IDトークン>` ヘッダにセット

リフレッシュトークンは Redis に保存（TTL 管理あり）。ID トークンは毎回リフレッシュから取得する。

### 新規 j-Quants エンドポイント追加手順

1. `stock_api_client.go` にメソッドをインターフェース追加
2. レスポンス型を `stock_api_response_info_models.go` に定義
3. `*StockAPIClientImpl` に実装を追加
4. `make mock` → `make di`
5. 対応する usecase と CLI コマンドを追加

### レート制限

大量銘柄をループ処理する場合は `time.Sleep` や `errgroup` で制御（既存の `newStockBrandDailyPrices` を参考に）。

---

## やってはいけないこと

- `gen_model/`, `gen_query/`, `pb/`, `di/wire_gen.go`, `mock/` 配下を手動編集
- `fmt.Errorf` でエラーラップ — 常に `github.com/pkg/errors`
- usecase 層以上で `gen_model` をインポート
- `infrastructure/` にビジネスロジックを書く
- DB 接続なしで `make gen` を実行（`_gorm_gen/main.go` は実スキーマを参照）
- `make di` なしで Wire プロバイダを追加
- テストで外部 API（j-Quants 等）を実際に呼び出す
- `gorm.DB` を直接クエリに使用（gen_query 層を経由すること）
- `local`/`dev` 以外の環境で gRPC リフレクションを有効化
- Raspberry Pi 向けクロスコンパイル済みバイナリ `entrypoint/cli/spr-cli` をコミット

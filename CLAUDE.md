# CLAUDE.md — stock-price-repository

## プロジェクト概要

Go 1.25 製の株価データ収集システム。**j-Quants API** を主軸に Yahoo Finance API からデータを取得し MySQL に保存する。REST API / gRPC API / CLI の3つの実行形態を持つが、**コアはデータ収集 CLI**。Raspberry Pi 上で常時稼働する想定。モジュールパス: `github.com/Code0716/stock-price-repository`。

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

### 6. テスト方針

- **全テスト**: テーブル駆動テスト形式
- `ENVCODE=unit`: ユニットテスト、Docker 不要
- `ENVCODE=e2e`: E2E テスト、Docker MySQL 必須
- **外部 API（j-Quants, Yahoo Finance, Slack）は必ずモック化**、実 API 呼び出し禁止
- E2E テスト: `helper.SetupTestDB(t)` でテスト専用 DB を作成、`helper.TruncateAllTables` でケース間クリーンアップ

### 7. DI 登録

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

# Repository Guidelines

## プロジェクト構成とモジュール

このリポジトリは、株価データを収集・提供する Go 製の Clean Architecture サービスです。ユースケースは `usecase/`、モデルは `models/`、リポジトリインターフェースは `repositories/`、外部接続や永続化の実装は `infrastructure/`（`database`、`gateway`、`cli`）に配置します。起動点は `entrypoint/api`、`entrypoint/grpc`、`entrypoint/cli` です。DI は `di/`、GORM 生成コードは `infrastructure/database/gen_query/`、protobuf 生成物は `pb/`、モックは `mock/` にあります。DB マイグレーションは `db-migrations/` サブモジュールで管理します。E2E テストは `test/e2e`、通常の単体テストは各パッケージ横の `*_test.go` に置きます。

## ビルド・テスト・開発コマンド

- `make up`: Docker Compose で MySQL、Redis、REST API、gRPC を起動します。
- `make api` / `make grpc-server`: Air のホットリロード付きでローカルサーバーを起動します。
- `make cli command=<name>`: CLI コマンドを実行します。例: `make cli command=update_stock_brands_v1`
- `make gen`: GORM クエリコードと `go generate` の生成物を更新します。
- `make di`: Wire の DI コードを再生成します。
- `make lint`: `golangci-lint` を実行します。
- `make test-unit`, `make test-e2e`, `make test`: 単体テスト、E2E テスト、または脆弱性チェックを含む全体テストを実行します。
- `make migrate-up` / `make migrate-down`: ローカル MySQL のマイグレーションを適用またはロールバックします。

## コーディング規約と命名

Go 標準の整形と import 整理（`gofmt` / `goimports`）を使います。パッケージ名は短く小文字にし、ディレクトリ名と揃えてください。`StockBrandRepository` や `CreateDailyStockPriceInteractor` のように、ドメイン上の役割が分かる名前を優先します。実装型は既存方針に合わせて `Impl` サフィックスを使います。生成ファイルは手編集しません。

## 実装上の重要ルール

エラーは `github.com/pkg/errors` の `errors.Wrap` / `errors.Errorf` で文脈を付けて返します。書き込みを行う repository は `GetTxQuery(ctx)` でトランザクションを引き継ぎ、トランザクション開始は usecase 層の `tx.DoInTx` に限定します。usecase 層は `infrastructure/` や `gen_model/` を直接 import せず、DB アクセスは repository インターフェース経由、外部 API は `gateway.StockAPIClient` 経由にしてください。モデル変換は repository 実装内の `convertToDomainModel` / `convertToDBModel` に閉じ込めます。

DB モデル、インターフェース、DI 対象を変えた場合は、原則として `make gen`、`make mock`、`make di` の順で再生成します。`make mock` は `mock/` を作り直すため、手編集したモックを残さないでください。

## テスト方針

Go 標準の `testing` パッケージを基本に、既存箇所では `testify` やプロジェクト内ヘルパーを使います。単体テストは対象パッケージの近くに置き、サービス横断や Docker 依存の検証は `test/e2e/` に追加します。通常の変更では `make test-unit`、DB・Redis・Docker・外部連携の挙動に関わる変更では `make test-e2e` も実行してください。

## コミットと Pull Request

直近の履歴では、`feat:` や `fix:` などの conventional commit 接頭辞に、簡潔な日本語または英語の説明を続けています。例: `feat: 決算カレンダーAPIを追加`、`fix: handle empty symbol prefix`

Pull Request のタイトル、本文、レビューコメントは日本語で記載します。本文には、課題と解決内容の要約、影響するコマンドや API、実行したテスト、関連 Issue を記載します。API 変更ではリクエスト・レスポンス例を添えてください。マイグレーションを含む場合は、番号と `db-migrations/schema/schema.sql` 更新有無を明記します。

## セキュリティと設定

`.env`、認証情報、API トークン、DB ダンプ、Box/Slack のシークレットはコミットしないでください。j-Quants、MySQL、Redis、Slack、Box の設定値は `readme.md` に記載された環境変数で管理します。`db-migrations/` と `proto-definitions/` は外部ソースとして扱い、各サブモジュールの手順に従って更新します。

## エージェント向け補足

このリポジトリでの説明、レビュー、提案は日本語で行います。より細かい層別ルールは、ルートの `CLAUDE.md` と各ディレクトリの `CLAUDE.md`（例: `usecase/CLAUDE.md`, `infrastructure/database/CLAUDE.md`, `infrastructure/gateway/CLAUDE.md`）も確認してください。

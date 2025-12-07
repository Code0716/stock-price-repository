````instructions
# GitHub Copilot Instructions for stock-price-repository

## 基本情報

### プロジェクト概要

Go 言語で構築された Clean Architecture 準拠の株価データ収集システム。
Yahoo Finance や j-Quants API から上場銘柄、日足、日経平均日足などを収集し、MySQL に保存する。

### 技術スタック

- **言語**: Go 1.25.0
- **CLI フレームワーク**: `github.com/urfave/cli/v2`
- **ORM**: `gorm.io/gorm`, `gorm.io/gen` (Type-safe Query Builder)
- **データベース**: MySQL (`github.com/go-sql-driver/mysql`)
- **依存性注入**: `github.com/google/wire`
- **Redis**: `github.com/redis/go-redis/v9`
- **ロギング**: `go.uber.org/zap`
- **エラーハンドリング**: `github.com/pkg/errors`
- **モック**: `go.uber.org/mock`

### ディレクトリ構造

```
stock-price-repository/
├── entrypoint/             # アプリケーションのエントリーポイント
│   └── cli/                # CLIコマンド定義
├── usecase/                # ビジネスロジック
├── repositories/           # リポジトリ・トランザクションインターフェース定義
├── models/                 # ドメインモデル
├── infrastructure/         # 外部インターフェースの実装
│   ├── database/           # DBアクセス実装 (GORM)
│   │   ├── gen_model/      # GORM Gen生成モデル
│   │   └── gen_query/      # GORM Gen生成クエリ
│   ├── gateway/            # 外部APIクライアント実装
│   └── cli/                # CLI実行基盤
├── driver/                 # 外部ライブラリ・サービスの初期化
├── di/                     # 依存性注入設定 (Wire)
├── mock/                   # モックファイル (mockgen生成)
└── _gorm_gen/              # GORM Gen生成ツール
```

## Go 言語コード生成・レビュー規則

### 基本原則

- **シンプルさ優先**: 賢さより明確さとシンプルさを重視
- **least surprise**: 予期しない動作を避ける
- **happy path 左寄せ**: インデントを最小限に、早期リターンでネストを減らす
- **ゼロ値活用**: 有効なゼロ値を持つ設計
- **idiomatic Go**: 慣用的な Go コードパターンに準拠

### 命名規則

#### パッケージ・変数・関数

- **パッケージ**: 小文字単語、単数形推奨。機能を表す名前。
- **変数・関数**: mixedCaps (camelCase)。エクスポートするものは大文字開始。
- **インターフェース**: `-er` サフィックス推奨。
- **定数**: エクスポートは PascalCase、非エクスポートは camelCase。

#### Repository 命名規則

- **メソッド名**: データアクセスパターンを明示（`Find`, `FindAll`, `Upsert`, `Delete`）。
- **主語**: メソッド名に主語を含めるかは文脈によるが、明確さを重視。
  - 例: `UpsertStockBrands`, `FindFromSymbol`

### エラーハンドリング

- `github.com/pkg/errors` を使用。
- `errors.Wrap(err, "message")` でスタックトレースとコンテキストを保持。
- 関数呼び出し直後のエラーチェック。
- エラー変数は `err` と命名。
- 正当な理由なしに `_` でエラー無視禁止。

### 並行性

- **Goroutine**: `sync.WaitGroup` や `errgroup` で管理し、リークを防止。
- **Context**: キャンセル信号の伝播に `context.Context` を使用。

### テスタビリティ設計

- **インターフェース依存**: 具象型ではなくインターフェースに依存させる。
- **DI**: `wire` を使用して依存関係を注入。
- **Mock**: `go.uber.org/mock` で生成したモックを使用。
- **時間・ランダム値**: インターフェース経由で注入し、テスト時に制御可能にする。

## アーキテクチャ層の責務

### Clean Architecture 層間の責務分離

- **Entrypoint (CLI)**:
  - CLI コマンドの引数解析、バリデーション。
  - UseCase の呼び出し。
  - エラーの表示。
- **UseCase**:
  - ビジネスロジックのオーケストレーション。
  - トランザクション境界の管理。
  - ドメインモデルの操作。
- **Repositories (Interface)**:
  - データアクセスの抽象化。
  - ドメインモデルの入出力。
- **Infrastructure**:
  - **Database**: GORM Gen を使用した DB 操作の実装。ドメインモデルと DB モデルの変換。
  - **Gateway**: 外部 API (Yahoo Finance, j-Quants) との通信実装。
- **Models**:
  - ドメインロジックとデータ構造。

## データベース操作 (GORM)

### GORM Gen の使用

- `infrastructure/database/gen_query` のクエリビルダを優先使用。
- 生 SQL は避け、Type-safe なクエリ構築を行う。
- ドメインモデル (`models`) と DB モデル (`infrastructure/database/gen_model`) は明確に分離し、リポジトリ層で変換を行う。

### トランザクション処理

- `infrastructure/database` 内の `GetTxQuery` パターンを使用し、コンテキスト経由でトランザクションを伝播させる。

```go
func (r *StockBrandRepositoryImpl) FindAll(ctx context.Context) ([]*models.StockBrand, error) {
    tx, ok := GetTxQuery(ctx)
    if !ok {
        tx = r.query
    }
    // ...
}
```

## テスト生成指針

### テスト構成

- **テーブル駆動テスト**: 各層のテストはテーブル駆動テスト (Table Driven Tests) で実装。
- **モックの使用**: `go.uber.org/mock` (mockgen) を使用。
- **テストファイル配置**: 実装ファイルと同じディレクトリに `_test.go` を配置。

### モック生成・管理

- **生成コマンド**: インターフェース定義ファイルに `//go:generate mockgen ...` を記述。
- **出力先**: `mock/` ディレクトリ配下の各パッケージディレクトリ。
  - 例: `mock/repositories/stock_brand.go`

### テストコード記述ルール

- **関数ごとのテスト作成**: テスト関数はテスト対象の関数ごとに作成する（1対1対応）。**プライベート関数（非公開関数）も含め、全ての関数に対してテストを作成する。**
- **網羅的なテストケース**: 正常系だけでなく、異常系、境界値、エッジケースを網羅するテストケースを作成する。
- **Mock の初期化**: `fields` 構造体の各フィールドは `func(ctrl *gomock.Controller) Interface` 型とし、テストケースごとに必要なモックのみを初期化する関数を定義する。
- **不要なモックの除外**: テストケースで使用しないリポジトリやサービスは `nil` (または未定義) とし、テスト実行時に `nil` チェックを行って設定する。これにより、テストのセットアップを最小限に保ち、可読性を向上させる。
- **未使用のモック定義の削除**: テストケース内で使用しないモックの初期化関数（`return nil` を返すだけのものなど）は記述せず、フィールド自体を省略する。
- **引数の厳密なチェック**: `gomock.Any()` の使用は避け、可能な限り具体的な値や `gomock.Eq()` を使用して引数を検証する。
- **アサーション**: `reflect.DeepEqual` や `github.com/stretchr/testify/assert` を使用して結果を検証する。

### E2Eテスト方針

- **配置場所**: `test/e2e` ディレクトリに配置する。
- **実行環境**: Docker上のMySQLを使用し、テスト実行ごとに専用のDBを作成・破棄する。
- **外部API**: `mock/gateway` を使用して外部API (Yahoo Finance, j-Quants) をモック化し、実際のAPIコールは行わない。
- **実行方法**: `infrastructure/cli.Runner` を使用してCLIコマンドの実行をシミュレートする。
- **ヘルパー利用**: DBセットアップなどの共通処理は `test/helper` パッケージを利用する。
- **テーブル駆動テスト**: E2Eテストもテーブル駆動テスト形式で記述する。
  - `setup` 関数でテストケースごとの初期データ投入を行う。
  - `check` 関数で実行後のDB状態検証を行う。
  - 各テストケース実行前に `helper.TruncateAllTables` でDBをクリーンアップする。

#### 推奨されるE2Eテストコード構成例

```go
func TestE2E_CommandName(t *testing.T) {
	// DBセットアップ
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// Redis, Mock等のセットアップ
	// ...

	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T)
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "正常系",
			args: args{
				cmdArgs: []string{"command", "subcommand"},
			},
			setup: func(t *testing.T) {
				// 初期データ投入
			},
			wantErr: false,
			check: func(t *testing.T) {
				// 結果検証
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テーブルクリア
			helper.TruncateAllTables(t, db)

			if tt.setup != nil {
				tt.setup(t)
			}

			// Runnerの構築と実行
			// ...
			err := runner.Run(context.Background(), tt.args.cmdArgs)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}
```

## Copilot の振る舞い設定

### 言語設定

- **回答・レビュー言語**: すべての応答、コードの説明、プルリクエストのレビューコメント、コミットメッセージの提案は **日本語** で行ってください。
- 英語で質問された場合でも、文脈から日本人の開発者であると判断できる場合は日本語で回答してください。
````

## gRPCアーキテクチャ

### proto定義の管理

- **別リポジトリ管理**: proto定義は `Code0716/stock-price-proto` で管理。
- **クローン方式**: `make proto-setup` で `proto-definitions/` にクローン。
- **gitignore**: `proto-definitions/` はコミット対象外。
- **生成コード**: `pb/` 配下の生成コードはコミット対象。

### コード生成フロー

1. `make proto-setup`: 初回のみ、proto定義リポジトリをクローン。
2. `make proto-pull`: 最新のproto定義を取得。
3. `make proto-gen`: `buf generate` でGoコードを `pb/` に生成。
4. `make mock`: モックの再生成（proto変更時）。

### gRPCサーバー構成

- **エントリーポイント**: `entrypoint/grpc/main.go`
- **サービス実装**: `entrypoint/grpc/server/stock_service_server.go`
- **ポート**: デフォルト50051（環境変数 `GRPC_PORT` で変更可能）
- **Reflection**: `APP_ENV=local` または `dev` 時のみ有効化。
- **依存性注入**: `di/wire.go` の `InitializeStockServiceServer` で構築。

### gRPCテスト方針

- **ユニットテスト**: サーバー実装（`*_server_test.go`）でモックを使用。
- **E2Eテスト**: `test/e2e/grpc_*_test.go` で実DBを使用した統合テスト。
- **CI/CD**: GitHub Actionsで自動的に `make proto-setup && make proto-gen` を実行。

### gRPCコーディング規則

- **エラーハンドリング**: `google.golang.org/grpc/status` でgRPCステータスコードを返す。
- **コンテキスト伝播**: 全てのRPCメソッドで `context.Context` を使用。
- **メッセージ変換**: ドメインモデルとprotobufメッセージの変換は各サーバー実装で行う。
- **時刻フォーマット**: RFC3339形式（`2006-01-02T15:04:05Z07:00`）で文字列化。

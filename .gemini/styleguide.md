# Stock Price Repository Style Guide

このプロジェクトは **Go (Golang)** で記述され、**Clean Architecture** を採用しています。開発にあたっては以下のガイドラインを遵守してください。

## 1. アーキテクチャとディレクトリ構造 (Clean Architecture)

プロジェクトは明確なレイヤー構造を持っています。依存関係は内側（Usecase, Domain）に向かうように設計してください。

- **`entrypoint/` (Interface Adapters)**:
  - 外部からの入力を受け取る層（CLI, gRPC, REST API）。
  - ビジネスロジック（Usecase）を呼び出し、レスポンスを返却します。
  - `main` パッケージを含み、DI（Dependency Injection）のルートとなります。
- **`usecase/` (Application Business Rules)**:
  - アプリケーション固有のビジネスロジック。
  - `Interactor` として実装され、リポジトリなどのインターフェースに依存します（実装には依存しない）。
- **`domain/` / `models/` (Enterprise Business Rules)**:
  - ドメインモデルやエンティティ。外部依存を持たない純粋なデータ構造とロジック。
- **`infrastructure/` (Frameworks & Drivers)**:
  - Usecase層で定義されたインターフェース（Repository等）の具体的な実装。
  - データベースアクセス、外部APIコールなどがここに含まれます。
- **`driver/`**:
  - DB接続、Redisクライアント、ロガーなどの低レベルなインフラストラクチャ設定・ラッパー。
- **`di/`**:
  - **Google Wire** (`github.com/google/wire`) を使用した依存解決の設定。

## 2. Go コーディング規約

- **フォーマット**: 標準の `gofmt` に加え、`goimports` を使用してください。
  - インポート順序: 標準ライブラリ -> サードパーティ -> プロジェクト内パッケージ（`github.com/Code0716/stock-price-repository/...`）。
- **エラーハンドリング**:
  - エラーは握りつぶさず、必ず処理するか呼び出し元に返してください。
  - `fmt.Errorf("context: %w", err)` を使用してエラーをラップし、コンテキスト（文脈）を付与してください。
- **Context**:
  - I/O操作（DB, API）を行う関数には、必ず第一引数として `context.Context` を渡してください。
- **Linter**:
  - `.golangci.yml` の設定に従い、`golangci-lint` をパスさせてください。

## 3. データベース (GORM & MySQL)

- **ORM**: `gorm.io/gorm` を使用しています。
- **モデル定義**: 構造体タグを使用してカラム定義やJSONマッピングを明確に記述してください。
- **トランザクション**: 複数の更新処理を行う場合は、必ずトランザクションを使用してください。
- **金額・数値**: 株価などの計算には精度誤差を防ぐため、適切な型（`DECIMAL`等）やライブラリの使用を検討してください。

## 4. テスト (Testing)

- **フレームワーク**: 標準の `testing` パッケージを使用。
- **アサーション**: `github.com/stretchr/testify/assert` を使用して可読性を高めてください。
- **モック**: `go.uber.org/mock` (gomock) を使用して、インターフェース（Repository等）のモックを生成・利用してください。
- **パターン**: **Table-Driven Tests** を強く推奨します。正常系と異常系のテストケースを網羅してください。
  - テストケース名は日本語で記述し、何を確認するテストなのか明確にしてください。

## 5. 生成AIエージェントへの指示

- コードを変更する際は、必ず周囲のコードや既存のテストを確認し、スタイルを統一してください。
- 新しい機能を追加する場合は、対応する単体テスト（Unit Test）も併せて作成してください。
- 複雑なロジックには、*Why*（なぜそうするのか）を説明するコメントを追加してください。
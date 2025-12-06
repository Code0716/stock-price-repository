# stock-price-repository

Go 言語で構築された Clean Architecture 準拠の株価データ収集システム。
Yahoo Finance や j-Quants API から上場銘柄、日足、日経平均日足などを収集し、MySQL に保存します。

本システムは **Raspberry Pi** 上での常時稼働を想定して設計されています。
また、本リポジトリの責務はあくまで「データの収集と保存」に限定されており、収集したデータを用いた株価分析やシミュレーションなどは、連携する別のコンテナ（システム）側で実行するアーキテクチャを採用しています。

## Features

- **株価データ収集**: 上場銘柄情報と日足株価を取得。
- **指数データ**: 日経平均 (Nikkei 225) と NY ダウ (DJI) のヒストリカルデータを収集。
- **REST API**: 収集した株価データを提供する REST API。
- **データエクスポート**: 保存されたデータを SQL ファイルとしてエクスポート。
- **トークン管理**: Redis を使用した j-Quants API リフレッシュトークンの管理。
- **Clean Architecture**: 保守性とテスト容易性を考慮した設計。

## Tech Stack

- **Language**: Go 1.25.0
- **CLI Framework**: `github.com/urfave/cli/v2`
- **Database**: MySQL 8.0
- **ORM**: GORM (`gorm.io/gorm`), GORM Gen (`gorm.io/gen`)
- **Cache**: Redis (`github.com/redis/go-redis/v9`)
- **Dependency Injection**: Google Wire (`github.com/google/wire`)
- **Logging**: Zap (`go.uber.org/zap`)
- **Testing**: `go.uber.org/mock` (Mockgen)

## Prerequisites

- **Go**: 1.25.0 以上
- **Docker**: MySQL と Redis コンテナの実行に必要
- **Make**: ビルドおよびユーティリティコマンドの実行に必要

## Installation / Setup

1. **リポジトリのクローン**

   ```bash
   git clone <repository-url>
   cd stock-price-repository
   ```

2. **環境とツールの初期化**
   必要な Go ツール (`wire`, `golangci-lint`, `air` 等) をインストールし、`.env` ファイルをセットアップします。

   ```bash
   make init
   ```

3. **インフラストラクチャの起動 (Docker)**
   MySQL と Redis コンテナを起動します。

   ```bash
   make up
   ```

4. **マイグレーションの実行**
   データベーススキーマを作成します。
   ```bash
   make migrate-up
   ```

## Configuration

`.env` ファイルで以下の環境変数を設定してください。

| Category     | Variable                                | Description                         |
| :----------- | :-------------------------------------- | :---------------------------------- |
| **App**      | `APP_ENV`                               | `local`, `dev`, `prod`              |
|              | `APP_LOG_LEVEL`                         | `debug`, `info` など                |
|              | `APP_TIMEZONE`                          | 例: `Asia/Tokyo`                    |
| **Database** | `STOCK_PRICE_REPOSITORY_MYSQL_HOST`     | DB ホスト                           |
|              | `STOCK_PRICE_REPOSITORY_MYSQL_DBNAME`   | DB 名                               |
|              | `STOCK_PRICE_REPOSITORY_MYSQL_USER`     | DB ユーザー                         |
|              | `STOCK_PRICE_REPOSITORY_MYSQL_PASSWORD` | DB パスワード                       |
| **Redis**    | `REDIS_HOST`                            | Redis ホスト (例: `localhost:6379`) |
| **APIs**     | `J_QUANTS_MAILADDRESS`                  | j-Quants ログインメールアドレス     |
|              | `J_QUANTS_PASSWORD`                     | j-Quants ログインパスワード         |
|              | `YAHOO_FINANCE_API_BASE_URL`            | Yahoo Finance API ベース URL        |
|              | `SLACK_NOTIFICATION_BOT_TOKEN`          | 通知用 Slack Bot トークン           |

## Usage (CLI Commands)

`make cli` コマンドを使用してアプリケーションを実行します。

### 株価銘柄の更新

j-Quants から最新の銘柄情報を取得し、DB に保存します。

```bash
make cli command=update_stock_brands_v1
```

### 当日の株価取得

全銘柄の当日の日足株価を取得します（市場クローズ後に実行）。

```bash
make cli command=create_daily_stock_price_v1
```

### ヒストリカル株価取得

全銘柄の過去の株価データを取得します。

```bash
make cli command=create_historical_daily_stock_price
```

### 指数ヒストリカルデータ取得

日経平均と NY ダウのヒストリカルデータを取得します。

```bash
make cli command=create_nikkei_and_dji_historical_data_v1
```

### j-Quants トークン更新

j-Quants のリフレッシュトークンを更新し、Redis に保存します。

```bash
make cli command=set_j_quants_api_token_to_redis_v1
```

### データエクスポート

DB のデータを SQL ファイルとしてエクスポートします。

```bash
make cli command=export_stock_brands_daily_stock_price_to_sql_v1
```

## Usage (API Server)

`make api` コマンドを使用して API サーバーを起動します（ホットリロード対応）。

```bash
make api
```

### エンドポイント

#### 日足株価取得

指定した銘柄の日足株価データを取得します。

- **URL**: `/daily-prices`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol` (必須): 銘柄コード (例: `1301`)
  - `from` (任意): 開始日 (YYYY-MM-DD)
  - `to` (任意): 終了日 (YYYY-MM-DD)

**Example Request:**

```bash
curl "http://localhost:8080/daily-prices?symbol=1301&from=2023-01-01&to=2023-01-31"
```

## Development

- **テスト実行**: `make test`
- **Lint 実行**: `make lint`
- **DI コード生成 (Wire)**: `make wire`
- **Mock 生成**: `make mock`
- **GORM コード生成**: `make gen`
- **コードフォーマット**: `make fmt` (要 `sql-formatter`)

## Architecture

Clean Architecture に基づいたディレクトリ構成になっています。

- **`entrypoint/`**: アプリケーションのエントリーポイント (CLI)。
- **`usecase/`**: ビジネスロジックとオーケストレーション。
- **`repositories/`**: リポジトリとトランザクションのインターフェース定義。
- **`infrastructure/`**: 外部インターフェースの実装。
  - **`database/`**: GORM を使用した DB 実装。
  - **`gateway/`**: 外部 API (Yahoo, j-Quants) クライアント。
- **`models/`**: ドメインエンティティ。
- **`config/`**: 設定読み込み。

かしこ。

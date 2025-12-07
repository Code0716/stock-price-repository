# stock-price-repository

Go 言語で構築された Clean Architecture 準拠の株価データ収集システム。
Yahoo Finance や j-Quants API から上場銘柄、日足、日経平均日足などを収集し、MySQL に保存します。

本システムは **Raspberry Pi** 上での常時稼働を想定して設計されています。
また、本リポジトリの責務はあくまで「データの収集と保存」に限定されており、収集したデータを用いた株価分析やシミュレーションなどは、連携する別のコンテナ（システム）側で実行するアーキテクチャを採用しています。

## Features

- **株価データ収集**: 上場銘柄情報と日足株価を取得。
- **指数データ**: 日経平均 (Nikkei 225) と NY ダウ (DJI) のヒストリカルデータを収集。
- **REST API**: 収集した株価データを提供する REST API。
- **gRPC API**: 高速な gRPC 通信による株価データ提供（高出来高銘柄の取得など）。
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
- **gRPC**: `google.golang.org/grpc`, `google.golang.org/protobuf`
- **Protocol Buffers**: proto 定義は別リポジトリ管理（`stock-price-proto`）
- **Buf**: protobuf コード生成ツール

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

4. **proto 定義のセットアップ（gRPC 使用時）**
   proto 定義をクローンし、gRPC コードを生成します。

   ```bash
   make proto-setup
   make proto-gen
   ```

5. **マイグレーションの実行**
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

#### 株価銘柄一覧取得

登録されている株価銘柄の一覧を取得します。

- **URL**: `/stock-brands`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol_from` (任意): 指定した銘柄コードより大きいもののみを取得 (最大 10 文字、英数字のみ)
  - `limit` (任意): 取得件数の上限 (1〜10000, デフォルト: 全件)
  - `only_main_markets` (任意): `true` を指定すると主要市場 (プライム・スタンダード・グロース) の銘柄のみ取得 (デフォルト: `false`)

**Example Requests:**

```bash
# 全銘柄を取得
curl "http://localhost:8080/stock-brands"

# 主要市場の銘柄のみ取得
curl "http://localhost:8080/stock-brands?only_main_markets=true"

# 銘柄コード "1301" より大きい銘柄を100件取得
curl "http://localhost:8080/stock-brands?symbol_from=1301&limit=100"

# 銘柄コード "2000" より大きい主要市場の銘柄を50件取得
curl "http://localhost:8080/stock-brands?symbol_from=2000&limit=50&only_main_markets=true"
```

**Response Example:**

```json
{
  "stock_brands": [
    {
      "ticker_symbol": "1301",
      "company_name": "極洋",
      "market_code": "111",
      "market_code_name": "プライム",
      "sector_33_code": "3050",
      "sector_33_name": "水産・農林業",
      "sector_17_code": "2",
      "sector_17_name": "食品"
    }
  ],
  "pagination": {
    "next_cursor": "1400",
    "limit": 100
  }
}
```

**Pagination:**

`limit` を指定した場合、レスポンスに `pagination` オブジェクトが含まれます。

- `next_cursor`: 次のページを取得するためのカーソル値（最後のページの場合は `null`）
- `limit`: 指定したリミット値

次のページを取得する場合は、`next_cursor` の値を `symbol_from` パラメータに指定してください。

```bash
# 最初のページ (100件取得)
curl "http://localhost:8080/stock-brands?limit=100"
# => { "stock_brands": [...], "pagination": { "next_cursor": "1400", "limit": 100 } }

# 次のページ (next_cursorを使用)
curl "http://localhost:8080/stock-brands?symbol_from=1400&limit=100"
# => { "stock_brands": [...], "pagination": { "next_cursor": "1500", "limit": 100 } }

# 最後のページ (next_cursorがnull)
curl "http://localhost:8080/stock-brands?symbol_from=9900&limit=100"
# => { "stock_brands": [...], "pagination": { "next_cursor": null, "limit": 100 } }
```

#### 日足株価取得

指定した銘柄の日足株価データを取得します。

- **URL**: `/daily-prices`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol` (必須): 銘柄コード (例: `1301`)
  - `from` (任意): 開始日 (YYYY-MM-DD)
  - `to` (任意): 終了日 (YYYY-MM-DD)
  - `sort` (任意): ソート順 (`asc` or `desc`, デフォルト: `asc`)

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

## gRPC Server

### Setup

1. **buf のインストール**

   ```bash
   go install github.com/bufbuild/buf/cmd/buf@latest
   ```

2. **proto 定義のクローンとコード生成**
   ```bash
   make proto-setup
   make proto-gen
   ```

### Running gRPC Server

#### ホットリロード付きでローカル起動（推奨）

```bash
make grpc-server
```

コードを変更すると自動的に再ビルド・再起動されます。

#### Docker Compose で起動

```bash
make grpc-server-docker
# または
docker compose up grpc-server
```

gRPC サーバーはポート `50051` で起動します。

#### 直接起動

```bash
APP_ENV=local GRPC_PORT=50051 go run entrypoint/grpc/main.go
```

環境変数 `GRPC_PORT` でポート番号を変更できます（デフォルト: 50051）。

### gRPC API Endpoints

#### GetHighVolumeStockBrands

高出来高銘柄を取得します。ページネーション対応。

- **Service**: `stock.StockService`
- **Method**: `GetHighVolumeStockBrands`
- **Request**: `GetHighVolumeStockBrandsRequest`
  - `symbol_from` (string, optional): カーソル（銘柄コード）。この値より大きい銘柄を取得。
  - `limit` (int32, optional): 取得件数。0 の場合は全件取得。
- **Response**: `GetHighVolumeStockBrandsResponse`

**Request Schema:**

```protobuf
message GetHighVolumeStockBrandsRequest {
  string symbol_from = 1;  // カーソル（銘柄コード）
  int32 limit = 2;         // 取得件数（0=全件）
}
```

**Response Schema:**

```protobuf
message GetHighVolumeStockBrandsResponse {
  repeated HighVolumeStockBrand brands = 1;
  PaginationInfo pagination = 2;  // limit > 0の場合のみ含まれる
}

message HighVolumeStockBrand {
  string stock_brand_id = 1;
  string ticker_symbol = 2;
  string company_name = 3;    // 銘柄名（stock_brandテーブルからJOIN）
  uint64 volume_average = 4;
  string created_at = 5;      // RFC3339形式 (例: "2024-01-15T10:30:00Z")
}

message PaginationInfo {
  string next_cursor = 1;  // 次のページのカーソル。空文字列の場合は最後のページ。
  int32 limit = 2;         // リクエストで指定したlimit値
}
```

**Example Request (grpcurl):**

```bash
# 全件取得
grpcurl -plaintext localhost:50051 stock.StockService/GetHighVolumeStockBrands

# 最初の10件を取得
grpcurl -plaintext -d '{"limit": 10}' localhost:50051 stock.StockService/GetHighVolumeStockBrands

# カーソル指定で次のページを取得（next_cursorが"7203"の場合）
grpcurl -plaintext -d '{"symbol_from": "7203", "limit": 10}' localhost:50051 stock.StockService/GetHighVolumeStockBrands
```

**Example Response:**

```json
{
  "brands": [
    {
      "stockBrandId": "550e8400-e29b-41d4-a716-446655440000",
      "tickerSymbol": "7203",
      "companyName": "トヨタ自動車",
      "volumeAverage": "15234567",
      "createdAt": "2024-01-15T10:30:00Z"
    },
    {
      "stockBrandId": "660e8400-e29b-41d4-a716-446655440001",
      "tickerSymbol": "6758",
      "companyName": "ソニーグループ",
      "volumeAverage": "12345678",
      "createdAt": "2024-01-15T10:31:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "9984",
    "limit": 10
  }
}
```

**Pagination の使い方:**

1. 最初のページを取得: `{"limit": 10}`
2. レスポンスの `pagination.nextCursor` を確認
3. 次のページを取得: `{"symbol_from": "<nextCursor>", "limit": 10}`
4. `pagination.nextCursor` が空文字列になるまで繰り返す

### Testing with grpcurl

開発環境（`APP_ENV=local`）では gRPC Reflection が有効になっているため、`grpcurl`で簡単にテストできます。

1. **grpcurl のインストール**

   ```bash
   brew install grpcurl
   ```

2. **サービス一覧の確認**

   ```bash
   grpcurl -plaintext localhost:50051 list
   ```

3. **メソッド一覧の確認**

   ```bash
   grpcurl -plaintext localhost:50051 list stock.StockService
   ```

4. **高出来高銘柄の取得**
   ```bash
   grpcurl -plaintext localhost:50051 stock.StockService/GetHighVolumeStockBrands
   ```

### Proto Definitions Management

proto 定義は別リポジトリ（[stock-price-proto](https://github.com/Code0716/stock-price-proto)）で管理されています。

#### proto 定義の更新手順

1. **最新の proto 定義を取得**

   ```bash
   make proto-pull
   ```

2. **コード再生成**

   ```bash
   make proto-gen
   ```

3. **モックの再生成**

   ```bash
   make mock
   ```

4. **テスト実行**
   ```bash
   make test
   ```

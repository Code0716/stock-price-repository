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
- **決算発表予定**: j-Quants から決算発表スケジュールを取得・保存。期間・銘柄フィルタ付き REST API で提供。
- **財務情報（業績）**: 売上高/営業利益/EPS/BPS など四半期推移データを取得・保存。REST API で提供。
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
   git clone --recurse-submodules <repository-url>
   cd stock-price-repository
   ```

   > すでにクローン済みの場合は submodule を初期化してください:
   > ```bash
   > git submodule update --init
   > ```

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

   マイグレーションファイルは `db-migrations/` submodule（[stock-price-db](https://github.com/Code0716/stock-price-db)）で管理されています。
   新しいマイグレーションを追加する場合は `stock-price-db` リポジトリで作業し、その後 submodule を更新してください:
   ```bash
   # stock-price-db で migration を追加・push した後
   git submodule update --remote db-migrations
   git add db-migrations
   git commit -m "chore: update db-migrations"
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
| **Box**      | `BOX_RCLONE_REMOTE_NAME`                | rclone のリモート名（デフォルト: `box`） |
|              | `BOX_RCLONE_FOLDER_PATH`                | アップロード先 Box フォルダパス（空欄でスキップ） |

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

### 決算発表予定の同期

j-Quants から近日の決算発表予定を取得し DB に保存します（毎日 19:00 過ぎに実行）。

```bash
make cli command=sync_fin_announcements
```

### 財務情報の同期

指定銘柄の財務情報（業績）を取得し DB に保存します（毎日 18:00 過ぎに実行）。

```bash
make cli command="sync_fin_statements --symbol=7203"
```

### データエクスポート

DB のデータを SQL ファイルとして mysqldump し、Box (box.com) へ自動アップロードします。

#### 年ごとの日足データをエクスポート

```bash
# 特定の年を指定
make export-yearly year=2024

# 年を省略すると DB 内の全年を自動検出して一括エクスポート
make export-yearly
```

対象テーブル: `stock_brands_daily_stock_prices`, `stock_brands_daily_stock_prices_for_analyze`
出力先: `STOCK_PRICE_REPOSITORY_MYSQL_BACKUP_PATH/{year}_{table}.sql`
年省略時は `stock_brands_daily_stock_prices` テーブルに存在する全年を検出してループ処理する。

#### マスターデータをエクスポート

```bash
make cli command=export_master_data
```

対象テーブル: `stock_brand`, `applied_stock_splits_history`, `nikkei_stock_average_daily_price`, `dji_stock_average_daily_stock_price`
出力先: `STOCK_PRICE_REPOSITORY_MYSQL_BACKUP_PATH/{YYYYMMDD}_{table}.sql`

> **Box アップロードについて**: SQL ファイル生成後、Box の設定フォルダへ自動アップロードされます。Box 環境変数が未設定の場合はアップロードをスキップし（警告ログのみ）、コマンドは正常終了します。Box のセットアップ手順は [Box セットアップ](#box-セットアップ) を参照してください。

## Usage (API Server)

Docker Compose で MySQL、Redis、API サーバーを起動します。API コンテナは Air を使用するため、ホスト側の Go ファイルを変更すると自動で再ビルド・再起動されます。

```bash
make api-docker
```

起動後、REST API は `http://localhost:8080` で利用できます。

```bash
curl "http://localhost:8080/stock-brands"
```

停止する場合は次のコマンドを実行します。

```bash
docker compose down
```

ホスト上に Air をインストール済みで、DB と Redis を別途起動している場合は、ローカルでも API サーバーを起動できます。

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

#### 決算発表予定一覧取得

近日の決算発表予定を取得します。

- **URL**: `/fin-announcements`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol` (任意): 銘柄コードで絞り込み
  - `from` (任意): 開始日 (YYYY-MM-DD)
  - `to` (任意): 終了日 (YYYY-MM-DD)
  - `page` (任意): ページ番号 (デフォルト: `1`)
  - `limit` (任意): 取得件数 (デフォルト: `20`, 最大: `100`)

**Example Request:**

```bash
# 今後2週間の発表予定を取得
curl "http://localhost:8080/fin-announcements?from=2026-05-16&to=2026-05-30"
```

**Response Example:**

```json
{
  "announcements": [
    {
      "id": "...",
      "tickerSymbol": "7203",
      "announcementDate": "2026-05-21",
      "fiscalYear": "3月31日",
      "fiscalQuarter": "本決算",
      "sector17Code": "",
      "sector33Code": ""
    }
  ],
  "pagination": { "page": 1, "limit": 20, "total": 42, "total_pages": 3 }
}
```

#### 次回決算発表日取得

指定銘柄の次回決算発表日を1件取得します。

- **URL**: `/fin-announcements/next`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol` (必須): 銘柄コード

**Example Request:**

```bash
curl "http://localhost:8080/fin-announcements/next?symbol=7203"
```

**Response Example:**

```json
{
  "announcement": {
    "id": "...",
    "tickerSymbol": "7203",
    "announcementDate": "2026-08-06",
    "fiscalYear": "3月31日",
    "fiscalQuarter": "1Q",
    "sector17Code": "",
    "sector33Code": ""
  }
}
```

#### デイトレード実現損益サマリー取得

SBI CSV でインポートしたデイトレード取引の損益サマリーを集計します。

- **URL**: `/daytrade/summary`
- **Method**: `GET`
- **Query Parameters**:
  - `granularity` (必須): 集計粒度 `daily` / `monthly` / `yearly` / `all`
  - `from` (任意): 開始日 (YYYY-MM-DD)
  - `to` (任意): 終了日 (YYYY-MM-DD)

**Example Request:**

```bash
curl "http://localhost:8080/daytrade/summary?granularity=monthly&from=2026-01-01&to=2026-05-31"
```

#### デイトレード銘柄別サマリー取得

- **URL**: `/daytrade/summary-by-symbol`
- **Method**: `GET`
- **Query Parameters**:
  - `from` (任意): 開始日 (YYYY-MM-DD)
  - `to` (任意): 終了日 (YYYY-MM-DD)

#### デイトレード約定一覧取得

指定日の約定履歴を取得します。

- **URL**: `/daytrade/executions`
- **Method**: `GET`
- **Query Parameters**:
  - `date` (必須): 約定日 (YYYY-MM-DD)

#### デイトレードデータ期間取得

インポート済みデータの最古・最新の約定日を返します。

- **URL**: `/daytrade/range`
- **Method**: `GET`

#### SBI CSV インポート

SBI 証券の信用取引約定履歴 CSV をインポートします。

- **URL**: `/daytrade/executions/import`
- **Method**: `POST`
- **Content-Type**: `multipart/form-data`
- **Form Fields**:
  - `file` (必須): SBI CSV ファイル（Shift-JIS エンコード）
  - `replace` (任意): `true` を指定すると既存の SBI インポートデータを全削除してから再挿入する（デフォルト: 増分インポート）

**Example Request:**

```bash
# 増分インポート（重複はスキップ）
curl -X POST http://localhost:8080/daytrade/executions/import \
  -F "file=@DOMESTIC_STOCK_20260523090406.csv"

# 置換インポート（既存 SBI データを全削除して再挿入）
curl -X POST http://localhost:8080/daytrade/executions/import \
  -F "file=@DOMESTIC_STOCK_20260523090406.csv" \
  -F "replace=true"
```

**Response Example:**

```json
{
  "inserted": 1128,
  "skipped":  0,
  "deleted":  1128
}
```

- `inserted`: 今回挿入した件数
- `skipped`: UNIQUE 制約の重複によりスキップした件数
- `deleted`: 置換モード時に削除した件数（通常インポート時は `null`）

---

### SBI CSV フォーマットについて

#### 新旧フォーマットの違い

SBI 証券の約定履歴 CSV は時期によってカラム順序が異なります。パーサは**ヘッダ行を動的に解析**するため、新旧どちらのフォーマットも自動的に処理します。

| カラム | 旧フォーマット | 新フォーマット（`DOMESTIC_STOCK_*`） |
|---|---|---|
| ファイル判別 | `信用` カラムあり | `口座` カラムあり |
| 取引種別 (`trade_kind`) | `取引` 列の値（例: `売建`） | 空文字（カラムなし） |
| 建玉種別 (`margin_kind`) | `信用` 列の値（例: `返済売`） | `取引` 列の値（例: `返済売`） |

#### 同一取引の重複識別 (`occurrence_no`)

同じ約定日・銘柄・価格・数量を複数同日約定した場合（部分約定の積み重ねや同一値段での複数約定）、UNIQUE キーが衝突します。これを識別するため `occurrence_no`（0 始まりの連番）を UNIQUE キーに含めています。

```
UNIQUE KEY: (executed_on, ticker_symbol, trade_kind, margin_kind,
             quantity, trade_amount, unit_price, profit_loss, occurrence_no)
```

#### SBI Web 表示と CSV の損益差異について

SBI の「信用取引損益照会」画面と約定履歴 CSV のエクスポートでは、実現損益が**数円〜十数円程度ずれることがあります**。これは SBI の仕様です。

確認済みの原因：
- **平均取得単価の精度差**: SBI 内部システムはより高精度で平均取得単価を管理しているが、CSV エクスポートでは小数点以下 0〜1 桁に丸めて出力される。この丸め差が多数の取引で積み重なる
- **集計基準の違い**: Web 画面の損益集計と CSV エクスポートで、端数処理や集計ロジックが微妙に異なる場合がある

このシステムは **SBI 公式の CSV（信用決済明細）の合計値と一致する値を表示**しています。

#### 制度信用 日計りの諸費用

一般信用の売り在庫がない場合に制度信用で売建てを行った取引では、**同日決済（日計り）でも貸株料・金利が 1 日分発生します**。これらの諸費用は CSV の `実現損益(税引前・円)` にあらかじめ差し引き済みの値として含まれているため、インポート後の損益集計に自動的に反映されます。

---

#### 財務情報取得

指定した銘柄の財務情報（業績）の直近 N 四半期分を取得します。

- **URL**: `/fin-statements`
- **Method**: `GET`
- **Query Parameters**:
  - `symbol` (必須): 銘柄コード
  - `limit` (任意): 取得件数 (デフォルト: `8`, 最大: `20`)

**Example Request:**

```bash
curl "http://localhost:8080/fin-statements?symbol=7203&limit=8"
```

**Response Example:**

```json
{
  "statements": [
    {
      "id": "...",
      "tickerSymbol": "7203",
      "disclosedDate": "2026-05-08",
      "typeOfCurrentPeriod": "FY",
      "netSales": "50684952000000",
      "operatingProfit": "3766216000000",
      "ordinaryProfit": null,
      "profit": "3848098000000",
      "earningsPerShare": "295.25",
      "bookValuePerShare": "3062.82",
      "forecastNetSales": null,
      "forecastOperatingProfit": null,
      "forecastProfit": null,
      "forecastEps": null
    }
  ]
}
```

## Box セットアップ

データエクスポートコマンド（`export_yearly_data`, `export_master_data`）は、SQL ファイル生成後に **rclone** 経由で Box へ自動アップロードします。Individual（個人）アカウントで動作します。

### 1. rclone のインストール（ラズパイ）

`apt install rclone` は古いバージョンが入るため、公式インストーラを使用してください。

```bash
curl https://rclone.org/install.sh | sudo bash
```

### 2. rclone の Box 設定（headless 環境の場合）

ラズパイはブラウザがないため、**SSH ポートフォワーディング**を使って Mac のブラウザで OAuth 認証を行います。Mac 側に rclone のインストールは不要です。

**手順 1: Mac から SSH トンネルを張る**

```bash
ssh -L 53682:localhost:53682 pi@raspberrypi
```

`-L 53682:localhost:53682` により、Mac の `127.0.0.1:53682` へのアクセスをラズパイの `127.0.0.1:53682`（rclone の OAuth コールバックポート）に転送します。

**手順 2: ラズパイ側（SSH セッション内）で rclone を設定**

```bash
rclone config
```

対話形式で以下を選択します:

- `n` → New remote
- name: `box`（デフォルトのリモート名）
- Storage type: `box` を選択（番号で指定）
- client_id, client_secret: 空欄のまま Enter（rclone 同梱のクライアントを使用）
- `Use auto config?` → **`y`**

ラズパイのターミナルに以下のような URL が表示されます:

```
If your browser doesn't open automatically go to the following link:
http://127.0.0.1:53682/auth?...
```

**手順 3: Mac のブラウザでその URL を開く**

URL をコピーして Mac のブラウザで開くと、SSH トンネル経由でラズパイの rclone に到達し、Box の認可画面に遷移します。承認後、Box がコールバックを返してラズパイの rclone がトークンを `~/.config/rclone/rclone.conf` に保存します。

ラズパイのターミナルに `Got code` などが表示されれば成功です。続けて `y → q` で終了します。

**疎通確認:**

```bash
rclone lsd box:
```

ルート直下のフォルダ一覧が表示されれば設定完了です。

### 3. 環境変数の設定

`.env` ファイルに以下を追記します:

```env
BOX_RCLONE_REMOTE_NAME=box
BOX_RCLONE_FOLDER_PATH=stock-backup/
```

- `BOX_RCLONE_REMOTE_NAME`: `rclone config` で設定したリモート名（デフォルト: `box`）
- `BOX_RCLONE_FOLDER_PATH`: Box 内のアップロード先フォルダパス。**空欄にするとアップロードをスキップ**（ローカル SQL ファイルは削除されません）

> **注**: 自前の Box Custom App（User Authentication / OAuth 2.0）を使う場合は、Developer Console の Redirect URI に `http://127.0.0.1:53682/` を追加し、`rclone config` の `--client-id` / `--client-secret` オプションを渡してください。rclone 同梱の client ID で問題なければ不要です。

### 4. 再認証（トークン失効時）

以下のいずれかに該当する場合は再認証が必要です。

- 60日以上 rclone を Box に対して使わず、refresh_token が失効した
- Box 側でパスワード変更 / 2FA 再設定 / アプリの認可取り消しを行った
- ラズパイの `~/.config/rclone/rclone.conf` が破損・消失した

#### パターン A: 既存リモートが残っている場合（推奨）

`rclone config reconnect` で OAuth トークンのみ再取得します。リモート名や設定は維持されます。

**手順 1: Mac から SSH トンネルを張る**

```bash
ssh -L 53682:localhost:53682 toshi@raspberrypi.local
```

**手順 2: ラズパイ側で再認証コマンドを実行**

```bash
rclone config reconnect box:
```

`Use auto config?` → **`y`** を選択すると、初回設定と同様に `http://127.0.0.1:53682/auth?...` のような URL が表示されます。

**手順 3: Mac のブラウザで URL を開いて承認**

承認するとトークンが `~/.config/rclone/rclone.conf` に上書き保存されます。

**疎通確認:**

```bash
rclone lsd box:
```

#### パターン B: rclone.conf が完全に消えている場合

[2. rclone の Box 設定（headless 環境の場合）](#2-rclone-の-box-設定headless-環境の場合) の手順を最初からやり直してリモートを新規作成してください。`.env` / crontab の環境変数はそのままで OK です。

#### トラブル: ポート 53682 が使用中エラー

```
listen tcp 127.0.0.1:53682: bind: address already in use
```

過去の `rclone authorize` プロセスが残っている可能性があります。以下で確認・停止します。

```bash
ss -tlnp | grep 53682            # PID を確認
kill <PID>                        # 停止
```

---

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

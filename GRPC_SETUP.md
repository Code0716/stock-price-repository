# gRPC実装セットアップガイド

## 概要

`high_volume_stock_brands`テーブルから全件取得するgRPC APIを実装しました。

## アーキテクチャ

- **proto定義**: 別リポジトリ (`Code0716/stock-price-proto`) で管理
- **生成コード**: `pb/` ディレクトリ（コミット対象）
- **proto-definitions**: ローカルにクローン（`.gitignore`で除外）

## セットアップ手順

### 1. proto専用リポジトリのGitHub作成

```bash
# GitHubで新しいリポジトリを作成
# Repository name: stock-price-proto
# Visibility: Public
# その他のオプションは選択せず、空のリポジトリを作成

# セットアップスクリプトを実行
/tmp/setup-proto-repo.sh
```

### 2. proto定義のクローンとコード生成

```bash
cd /Users/toshiyukisugai/WEB/dev/stock-price-repository

# proto定義をクローン
make proto-setup

# コード生成
make proto-gen

# モック生成
make mock
```

### 3. テスト実行

```bash
# すべてのテストを実行
make test

# 個別テスト
go test -v ./usecase -run TestGetHighVolumeStockBrandsInteractor_Execute
go test -v ./entrypoint/grpc/server -run TestStockServiceServer_GetHighVolumeStockBrands
ENVCODE=e2e go test -v ./infrastructure/database -run TestHighVolumeStockBrandRepositoryImpl
```

## gRPCサーバー起動

### 方法1: ローカルで起動

```bash
# 環境変数を設定
export APP_ENV=local
export GRPC_PORT=50051

# サーバー起動
go run entrypoint/grpc/main.go
```

### 方法2: Dockerで起動

```bash
# gRPCサーバーをビルド・起動
docker compose up grpc-server

# バックグラウンドで起動
docker compose up -d grpc-server

# ログ確認
docker compose logs -f grpc-server
```

## 動作確認

### grpcurlでテスト

```bash
# grpcurlをインストール（初回のみ）
brew install grpcurl

# サービス一覧を確認
grpcurl -plaintext localhost:50051 list

# メソッド一覧を確認
grpcurl -plaintext localhost:50051 list stock.StockService

# 高出来高銘柄を取得
grpcurl -plaintext localhost:50051 stock.StockService/GetHighVolumeStockBrands
```

### テストデータの投入（オプション）

```bash
mysql -h 127.0.0.1 -u root -proot stock_price_repository << "SQL"
-- 銘柄を追加
INSERT INTO stock_brand (id, ticker_symbol, name, market_code, market_name, created_at, updated_at)
VALUES
  (UUID(), '1001', 'Test Brand 1', '111', 'Prime', NOW(), NOW()),
  (UUID(), '1002', 'Test Brand 2', '111', 'Prime', NOW(), NOW());

-- 高出来高銘柄を追加
INSERT INTO high_volume_stock_brands (stock_brand_id, ticker_symbol, volume_average, created_at)
SELECT id, ticker_symbol, 1000000, NOW() FROM stock_brand WHERE ticker_symbol = '1001'
UNION ALL
SELECT id, ticker_symbol, 2000000, NOW() FROM stock_brand WHERE ticker_symbol = '1002';
SQL
```

## proto定義の更新

proto定義を変更する場合：

```bash
# 1. stock-price-proto リポジトリで変更
cd /path/to/stock-price-proto
# proto/stock_service.proto を編集
git add .
git commit -m "Update proto definitions"
git push

# 2. このリポジトリで最新を取得
cd /Users/toshiyukisugai/WEB/dev/stock-price-repository
make proto-pull
make proto-gen
make mock

# 3. テスト実行
make test
```

## 実装ファイル一覧

### ドメイン層
- `models/high_volume_stock_brand.go` - ドメインモデル

### リポジトリ層
- `repositories/high_volume_stock_brand.go` - インターフェース
- `infrastructure/database/high_volume_stock_brand.go` - 実装
- `infrastructure/database/high_volume_stock_brand_test.go` - テスト

### ユースケース層
- `usecase/get_high_volume_stock_brands.go` - ビジネスロジック
- `usecase/get_high_volume_stock_brands_test.go` - テスト

### gRPCサーバー層
- `entrypoint/grpc/main.go` - エントリーポイント
- `entrypoint/grpc/server/stock_service_server.go` - サービス実装
- `entrypoint/grpc/server/stock_service_server_test.go` - テスト

### 生成コード
- `pb/stock_service.pb.go` - protobuf生成コード
- `pb/stock_service_grpc.pb.go` - gRPC生成コード

### 設定ファイル
- `di/wire.go` - 依存性注入設定
- `Dockerfile.grpc` - gRPCサーバー用Dockerfile
- `compose.yml` - Docker Compose設定
- `Makefile` - proto管理コマンド
- `.gitignore` - proto-definitions除外

## トラブルシューティング

### proto-definitionsが見つからない

```bash
make proto-setup
```

### 生成コードが古い

```bash
make proto-pull
make proto-gen
```

### モックが見つからない

```bash
make mock
```

### テストが失敗する

```bash
# データベースとRedisが起動しているか確認
docker compose ps

# マイグレーションを実行
make migrate-up

# 再テスト
make test
```

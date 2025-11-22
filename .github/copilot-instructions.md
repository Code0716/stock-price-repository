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
├── repositories/           # リポジトリインターフェース定義
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

- **Mock の初期化**: `fields` 構造体の各フィールドは `func(ctrl *gomock.Controller) Interface` 型とし、テストケースごとに必要なモックのみを初期化する関数を定義する。
- **不要なモックの除外**: テストケースで使用しないリポジトリやサービスは `nil` (または未定義) とし、テスト実行時に `nil` チェックを行って設定する。これにより、テストのセットアップを最小限に保ち、可読性を向上させる。
- **未使用のモック定義の削除**: テストケース内で使用しないモックの初期化関数（`return nil` を返すだけのものなど）は記述せず、フィールド自体を省略する。
- **アサーション**: `reflect.DeepEqual` や `github.com/stretchr/testify/assert` を使用して結果を検証する。

#### 推奨されるテストコード構成例

```go
func TestService_Method(t *testing.T) {
	type fields struct {
		// 必須の依存関係
		mainRepo func(ctrl *gomock.Controller) repository.MainRepository
		// オプショナルな依存関係（テストケースによって使わないもの）
		subRepo  func(ctrl *gomock.Controller) repository.SubRepository
	}
	type args struct {
		ctx context.Context
		id  uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *model.Result
		wantErr bool
	}{
		{
			name: "正常系 - 必要なモックのみ定義",
			fields: fields{
				mainRepo: func(ctrl *gomock.Controller) repository.MainRepository {
					mock := mockrepository.NewMockMainRepository(ctrl)
					mock.EXPECT().Find(gomock.Any(), gomock.Eq(uint64(1))).Return(&model.Entity{}, nil)
					return mock
				},
				// subRepo はこのテストケースでは使われないため定義しない（nil）
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			want:    &model.Result{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// 必須フィールドの初期化
			s := &service{
				mainRepo: tt.fields.mainRepo(mockCtrl),
			}

			// オプショナルフィールドの初期化（nilチェック）
			// テストケースで定義されていない場合はセットアップしない
			if tt.fields.subRepo != nil {
				s.subRepo = tt.fields.subRepo(mockCtrl)
			}

			got, err := s.Method(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Method() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Method() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

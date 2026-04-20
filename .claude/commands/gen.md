# /gen

コード生成を正しい順序で全て実行する。

モデル・インターフェース・DB スキーマを変更した後に使う。

## 実行内容

以下を順番に実行してください（並列実行禁止、順序依存あり）:

```bash
# 1. GORM Gen — DB スキーマから gen_model/ と gen_query/ を再生成
#    ※ MySQL が起動している必要がある（docker compose up で起動）
make gen

# 2. mockgen — //go:generate ディレクティブを元に mock/ を全削除して再生成
make mock

# 3. Wire — di/wire.go を元に di/wire_gen.go を再生成
make di
```

## エラーが出た場合

- `make gen` 失敗: MySQL が起動しているか確認（`docker compose up -d`）
- `make mock` 失敗: インターフェース定義ファイルに `//go:generate mockgen ...` ディレクティブがあるか確認
- `make di` 失敗: `di/wire.go` の Set に必要なプロバイダが全て登録されているか確認。コンパイルエラーメッセージを読んで不足しているプロバイダを追加する

## 生成ファイルは手動編集禁止

`gen_model/`, `gen_query/`, `mock/`, `di/wire_gen.go` は自動生成ファイル。次回 gen 実行時に上書きされる。

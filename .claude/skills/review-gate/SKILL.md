---
name: review-gate
description: git commit / gh pr create の前に必ず通すAIレビューゲート。コミット・PR作成の依頼時、またはフックが「AIレビュー未実施」で拒否した時に使う。
---

# review-gate

`.claude/hooks/review-gate.sh`（PreToolUse、`.claude/settings.json` で登録）が、レビュー済みの印が無い `git commit` / `gh pr create` を拒否する。この skill は印を付けるまでの手順。印は `.git/review-gate/` に置かれ、コミット対象外。

## コミット前
1. 変更を `git add`（**commit と同一コマンドに連結しない**）。`-a` を使う場合も可。
2. `Skill` で `code-review` を呼ぶ（既定 `medium`。対象はステージ済みdiff）。
3. `important` 指摘は修正する。修正したら再度 `git add` し、2 に戻って再レビュー（diffが変わると印は無効）。
4. 未解決の important が無ければ印を付ける:
   `bash "$CLAUDE_PROJECT_DIR"/.claude/hooks/review-gate.sh mark commit`（`git commit -a` 時は `commit-all`）
5. 通常どおり `git commit` を実行。

## PR作成前
1. 全コミット済みであること。`Skill` で `code-review` をブランチ全体（base との差分）に対して実行。
2. 指摘解消後 `bash "$CLAUDE_PROJECT_DIR"/.claude/hooks/review-gate.sh mark pr` → `gh pr create`。

## 注意
- `mark` の種別は実際のコミット形式と一致させる（`git commit -a` → `commit-all`、それ以外 → `commit`）。不一致だとハッシュが合わず拒否される。
- `git commit <path>` / `--only(-o)` / `--include(-i)` は対象diffが特定できないため使わない（`-o`/`-i` 系は拒否、index が空で未ステージ変更がある場合の path 指定も拒否。index に別の変更がある状態での path 指定は検知できない）。必ず `git add` してから素の `git commit`。
- `gh pr create --base <b>` を使う場合は `PR_BASE=<b> bash "$CLAUDE_PROJECT_DIR"/.claude/hooks/review-gate.sh mark pr`（レビューも `origin/<b>` との差分に対して行う）。
- commit と `gh pr create` を `&&` で連結しない（拒否される）。
- base 解決に失敗して PR が拒否されたら `git fetch origin` してから再実行。

## このゲートの限界（ソフトゲート）
- 目的は「Claude がレビューを忘れること」の防止で、意図的な迂回の完全防止ではない。コマンド文字列を正規表現で解釈するため、`{ git commit; }`・`git "commit"`・サブシェル内 cd・同一コマンド内の複数 commit などの特殊な書き方は検知できない/先頭だけ判定になる場合がある。
- 印は `mark` を呼んだ事実しか証明しない（レビュー実行までは検証しない）。
- 印は期限なしで `.git/review-gate/` に残る。溜まったら手で削除してよい。
- `cd`/`-C` 先が git リポジトリでないと拒否される。cd と commit は別コマンドに分ける。

## 禁止事項
- レビューを実行せず、または指摘が残ったまま `mark` しない。
- 印ファイル（`.git/review-gate/`）を手で作らない。
- ゲートを迂回するコマンド分割・別経路でのコミットをしない。迂回が必要ならユーザーに確認する。

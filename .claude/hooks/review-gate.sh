#!/bin/bash
# AIレビュー未実施の git commit / gh pr create を deny する PreToolUse フック(spr 専用)。
# 仕組み: 対象diffのハッシュ印($(git rev-parse --git-common-dir)/review-gate/<hash>)が無ければ拒否。
# 印は skill review-gate が、レビュー指摘ゼロ確認後に `review-gate.sh mark` で付ける。
# 使い方(skill用): review-gate.sh mark|hash commit|commit-all|pr
EMPTY_HASH="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

state_dir() { echo "$(cd "$(git rev-parse --git-common-dir)" && pwd)/review-gate"; }

# 失敗(base不明など)は非0を返す。パイプ内returnで握り潰されないよう先に変数へ取る。
diff_hash() { # $1=commit|commit-all|pr
  local out base mb
  case "$1" in
    commit)     out=$(git diff --cached) || return 1 ;;
    commit-all)
      # 初回コミット(HEADなし)は空treeとの差分
      if git rev-parse -q --verify HEAD >/dev/null 2>&1; then out=$(git diff HEAD) || return 1
      else out=$(git diff 4b825dc642cb6eb9a060e54bf8d69288fbee4904) || return 1; fi ;;
    pr)
      base=$(git symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null || echo origin/main)
      [ -n "$PR_BASE" ] && base="origin/$PR_BASE"   # gh pr create --base/-B 指定を反映
      mb=$(git merge-base "$base" HEAD 2>/dev/null) || return 1
      out=$(git diff "$mb"..HEAD) || return 1 ;;
    *) return 1 ;;
  esac
  printf '%s' "$out" | shasum -a 256 | cut -d' ' -f1
}

# --- skill 用サブコマンド ---
if [ "$1" = "hash" ] || [ "$1" = "mark" ]; then
  h=$(diff_hash "${2:-commit}") || { echo "diff取得に失敗(baseブランチ不明?)。git fetch origin を試してください" >&2; exit 1; }
  if [ "$1" = "hash" ]; then echo "$h"; else
    d=$(state_dir); mkdir -p "$d"; date +%FT%T > "$d/$h"; echo "marked $h"
  fi
  exit 0
fi

# --- フック本体 ---
PR_BASE=""   # 環境変数の PR_BASE はフックでは使わない（mark 時の手動指定専用）
deny() {
  if command -v jq >/dev/null 2>&1; then
    jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
  else
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"review-gate: %s"}}\n' "$1"
  fi
  exit 0
}

input=$(cat)
[ -z "$input" ] && exit 0
# 依存コマンドが無い場合、git/gh を含まない無関係なBash呼び出しまで止めないよう素通しにする
command -v jq >/dev/null 2>&1 || {
  printf '%s' "$input" | grep -Eq 'git|gh' || exit 0
  deny "jq が無いためレビューゲートを評価できません。jq をインストールしてください。"
}
cmd=$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)
[ -z "$cmd" ] && exit 0

command -v perl >/dev/null 2>&1 || {
  printf '%s' "$cmd" | grep -Eq 'git|gh' || exit 0
  deny "perl が無いためレビューゲートを評価できません。"
}
SCAN_PL="$(dirname "$0")/review-gate-scan.pl"
[ -r "$SCAN_PL" ] || deny "review-gate-scan.pl が見つからないためレビューゲートを評価できません。"

# 判定用: heredoc本体と引用符内(複数行含む)を除去（メッセージ中の文言での誤検知を防ぐ）
scan=$(printf '%s\n' "$cmd" | perl "$SCAN_PL" strip) || deny "コマンドの解析に失敗しました。"
# bash -c / eval 等のラッパー経由は引用符内が本体なので、除去せず引用符を空白化して走査(過検知側に倒す)。
# 判定は引用符除去後の文字列で行う（コミットメッセージ中の語に反応しない）
if printf '%s' "$scan" | grep -Eq '(^|[;&|( ])(bash|sh|zsh)( +-[a-zA-Z]+)* +-[a-zA-Z]*c( |$)|(^|[;&|( ])eval( |$)'; then
  scan=$(printf '%s\n' "$cmd" | perl "$SCAN_PL" wrap) || deny "コマンドの解析に失敗しました。"
  wrapped=1
fi

PRE='(^|[;&|(`!]|\$\(|command +|exec +|xargs +|-c +|env +|sudo +|time +|nohup +|nice +|then +|do +|else +|[A-Za-z_][A-Za-z0-9_]*=[^ ]+ +) *'
GITOPT='( +(-C +[^ ]+|-[^ ]+|[^ -][^ ]*=[^ ]*))*'
kind=""; has_commit=0; has_pr=0
echo "$scan" | grep -Eq "${PRE}git${GITOPT} +commit( |\$)" && { kind=commit; has_commit=1; }
echo "$scan" | grep -Eq "${PRE}gh( +(-R +[^ ]+|-[^ ]+))* +pr +create( |\$)" && { kind=pr; has_pr=1; }
[ -z "$kind" ] && exit 0
# commit と PR作成の連結は、PR判定時点で新コミットが存在しないため分離させる
[ "$has_commit$has_pr" = 11 ] && \
  deny "git commit と gh pr create を同一コマンドに連結しないでください。commit を先に実行し、PR 用に再度レビューしてから gh pr create してください。"

if [ "$kind" = commit ]; then
  # add と commit の連結は、フック時点のdiffが実際とズレるので分離させる
  echo "$scan" | grep -Eq "${PRE}git${GITOPT} +(add|stage)( |\$)" && \
    deny "git add と git commit を同一コマンドに連結しないでください。add を先に実行し、review-gate skill でレビュー後に commit してください。"
  # オプション判定は `git ... commit` と同じコマンド区間だけを見る（ls -la 等の無関係な -a を拾わない）
  seg=$(echo "$scan" | grep -Eo "git${GITOPT} +commit[^;&|]*" | head -1)
  echo "$seg" | grep -Eq ' (--only|--include|-[a-zA-Z]*[io][a-zA-Z]*)( |$)' && \
    deny "git commit --only/-o / --include/-i は対象diffを特定できないため使えません。git add 後に通常の git commit か -a を使ってください。"
  echo "$seg" | grep -Eq ' (-[a-zA-Z]*a[a-zA-Z]*|--all)( |$)' && kind=commit-all
fi

# 実行ディレクトリ: `cd <dir>` と `git -C <dir>` を反映（引用符・空白入りパス対応）。入れなければ fail closed
# bash -c "cd X && git commit" のようなラッパー内の cd も拾うため、ラッパー時は引用符を空白化済みの $scan から抽出する
if [ -n "$wrapped" ]; then dirs=$(printf '%s\n' "$scan" | perl "$SCAN_PL" dir); else dirs=$(printf '%s\n' "$cmd" | perl "$SCAN_PL" dir); fi \
  || deny "コマンドの解析に失敗しました。"
moved=""
while IFS= read -r line; do
  [ -z "$line" ] && continue
  dir="${line#*:}"; dir="${dir/#\~/$HOME}"
  cd "$dir" 2>/dev/null || deny "対象ディレクトリ($dir)に入れず、レビュー対象を特定できません。"
  moved=1
done <<< "$dirs"
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  # cd/-C した先が git 管理外なら、別リポジトリ扱いにせず拒否（(cd /tmp && ..); git commit 等の誤認含め fail closed）
  [ -n "$moved" ] && deny "cd / -C 先($dir)が git リポジトリではなく、レビュー対象を特定できません。cd を別コマンドに分けてから実行してください。"
  exit 0
fi
if [ "$kind" = pr ]; then
  # 本文・タイトル中の -B 等に引きずられないよう、まず引用符除去後のコマンド本体から取り、無ければ引用符付き指定のみ生コマンドから取る
  PR_BASE=$(echo "$scan" | sed -nE 's/.* (--base|-B)[ =]([A-Za-z0-9._\/-]+).*/\2/p' | head -1)
  [ -z "$PR_BASE" ] && PR_BASE=$(echo "$cmd" | sed -nE "s/.* (--base|-B)[ =][\"']([A-Za-z0-9._\/-]+)[\"'].*/\2/p" | head -1)
  echo "$PR_BASE" | grep -Eq '^[A-Za-z0-9._/-]*$' || PR_BASE=""
fi

h=$(diff_hash "$kind") || deny "PR の base ブランチ(origin/HEAD か origin/main)が解決できず、レビュー対象diffを特定できません。git fetch origin 後に再実行してください。"
[ -e "$(state_dir)/$h" ] && exit 0
# 空diff(amendのメッセージ変更等。PRはbase差分なし)はレビュー対象なし。
# ただし commit で index が空なのに未ステージ変更がある場合は、`git commit <path>` で未レビュー内容が入り得るので拒否
if [ "$h" = "$EMPTY_HASH" ]; then
  # --amend(メッセージ修正)/--allow-empty は index 以外を取り込まないので対象外
  if [ "$kind" = commit ] && ! echo "$seg" | grep -Eq -- '--amend|--allow-empty' && ! git diff --quiet 2>/dev/null; then
    deny "ステージ済みの変更が無いのに未ステージの変更があります（git commit <path> 等で未レビュー内容が入る恐れ）。git add 後に review-gate skill でレビューし、素の git commit を実行してください。"
  fi
  exit 0
fi
deny "AIレビュー未実施です（${kind} の diff にレビュー済みの印がありません）。Skill review-gate を実行し、レビュー指摘を解消して印を付けてから再実行してください。レビュー後にdiffを変更した場合は再レビューが必要です。"

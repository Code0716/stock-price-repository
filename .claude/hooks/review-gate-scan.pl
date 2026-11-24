#!/usr/bin/perl
# review-gate.sh 用の前処理。stdin のコマンド全文を処理して stdout に出す。
#   strip: heredoc本体と引用符内(複数行含む)を除去（メッセージ中の文言での誤検知を防ぐ）
#   wrap : heredoc本体のみ除去し引用符は空白化（bash -c 等ラッパー用。過検知側に倒す）
#   dir  : 実行ディレクトリ指定を "cd:<dir>" / "C:<dir>" で出力（引用符・空白入りパス対応、最後の指定を採用）
use strict; use warnings;
my $mode = shift // 'strip';
local $/;
my $s = <STDIN>;
$s = '' unless defined $s;

# 行継続(バックスラッシュ+改行)は1論理行に連結（`git commit \` の次行のオプションを見落とさない）
$s =~ s/\\\n/ /g;

# heredoc 本体を除去（here-string <<< は対象外）。マーカー行の残り($3)は保持して以降も走査する
$s =~ s/(?<!<)<<(?!<)-?[ \t]*(["']?)(\w+)\1([^\n]*)\n.*?\n[ \t]*\2[ \t]*(?:\n|\z)/$3\n/gs;

my $QUOTED = qr/"(?:[^"\\]|\\.)*"|'[^']*'/;   # 左から順に1パスで処理(二重/単一の入れ子誤認を防ぐ)

if ($mode eq 'dir') {
  # 引用部分をプレースホルダ化 → コマンド構造だけで cd / git -C を探す → 値は復元（メッセージ中の cd 等を誤認しない）
  my @q;
  $s =~ s/($QUOTED)/push @q, $1; "\x01" . $#q . "\x01"/ge;
  my $unq = sub {
    my $t = shift;
    $t =~ s/\x01(\d+)\x01/$q[$1]/g;
    $t =~ s/^(["'])(.*)\1$/$2/s;
    $t;
  };
  my $ARG = qr/\x01\d+\x01|[^\s;&|\x01]+/;
  my ($cd, $cc);
  while ($s =~ /(?:^|[;&|(])\s*cd\s+(?:--\s+)?($ARG)/gm) { $cd = $unq->($1); }
  # git -C はサブコマンドより前のオプション列にあるものだけ（git commit -C HEAD は対象外）
  while ($s =~ /\bgit((?:\s+(?:-C\s+$ARG|-c\s+$ARG|--[^\s;&|]+|-[^\s;&|]+))*)/g) {
    my $opts = $1;
    while ($opts =~ /-C\s+($ARG)/g) { $cc = $unq->($1); }
  }
  print "cd:$cd\n" if defined $cd;
  print "C:$cc\n"  if defined $cc;
} elsif ($mode eq 'wrap') {
  $s =~ tr/"'/  /;
  print $s;
} else {
  $s =~ s/$QUOTED//gs;
  print $s;
}

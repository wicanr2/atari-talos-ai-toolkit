#!/bin/sh
set -eu

# 從 trace.sh 產生的 trace 裡挑出指定位址第一次執行時的 cycle 數，
# 一行一個位址，方便直接對 Atari Talos 的 machine.Clocks。
#
# 用法：cycles.sh <trace.txt> <位址…>     位址寫成八位小寫十六進位，例如 00fc0088

TRACE=${1:?用法：cycles.sh <trace.txt> <位址…>}
shift
[ $# -gt 0 ] || { echo "至少要給一個位址" >&2; exit 1; }

for address in "$@"; do
  line=$(grep -m1 "video_cyc=.*: $address " "$TRACE" || true)
  if [ -z "$line" ]; then
    echo "\$$address	（未執行到）"
  else
    cycles=$(echo "$line" | sed -n 's/.*video_cyc= *\([0-9]*\).*/\1/p')
    text=$(echo "$line" | sed 's/.*: [0-9a-f]* //')
    echo "\$$address	$cycles	$text"
  fi
done

#!/bin/sh
set -eu

# 用 Hatari 取一份可重跑的 CPU trace，當作 Atari Talos 的外部 oracle。
#
# Hatari 只作 oracle：這裡跑的是上游未修改的二進位，兩邊各自執行同一顆 ROM，只比對
# 輸出的數字。不翻譯、不移植、不連結它的程式碼（AGENTS.md）。
#
# 為什麼是 --trace 而不是 --parse：debugger 那條路要互動式 stdin，中斷點觸發後 Hatari
# 會停在提示等輸入，在容器裡既不會結束也拿不到輸出。`--run-vbls` 加 `--trace-file`
# 是純輸出、會自己結束，可以無人值守重跑。
#
# trace 的每條指令長這樣（cycle 數是本檔最重要的欄位，用來對 Atari Talos 的 clocks）：
#
#   cpu video_cyc=   220 220@  0 220 : 00fc0088 4e70   reset
#
# 用法：trace.sh <TOS ROM> <輸出目錄> [VBL 數，預設 1]

ROM=${1:?用法：trace.sh <TOS ROM> <輸出目錄> [VBL 數]}
OUT=${2:?}
VBLS=${3:-1}
IMAGE=${HATARI_ORACLE_IMAGE:-atari-talos-hatari:2.4.1}

docker image inspect "$IMAGE" >/dev/null 2>&1 || {
  echo "找不到映像 $IMAGE。先建置（需要網路）：" >&2
  echo "  docker build --network default -t $IMAGE $(dirname "$0")/" >&2
  exit 1
}

ROM=$(cd "$(dirname "$ROM")" && pwd)/$(basename "$ROM")
mkdir -p "$OUT"
OUT=$(cd "$OUT" && pwd)
rm -f "$OUT/trace.txt"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e HOME=/tmp -e SDL_VIDEODRIVER=dummy -e SDL_AUDIODRIVER=dummy \
  -v "$ROM:/tos.img:ro" -v "$OUT:/out" \
  "$IMAGE" \
  --tos /tos.img --machine st --memsize 1 --sound off --fast-boot false \
  --run-vbls "$VBLS" --trace cpu_disasm,cpu_regs --trace-file /out/trace.txt \
  >"$OUT/hatari.log" 2>&1

[ -s "$OUT/trace.txt" ] || {
  echo "trace 是空的，看 $OUT/hatari.log" >&2
  exit 1
}
echo "trace：$OUT/trace.txt（$(wc -l < "$OUT/trace.txt") 行）"
echo "ROM SHA-256：$(sha256sum "$ROM" | cut -d' ' -f1)"

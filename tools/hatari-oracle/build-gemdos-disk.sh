#!/bin/sh
set -eu

if [ "$#" -ne 6 ]; then
    echo "用法：$0 OUTPUT.st START.PRG START.PAK GRAPHICS.DAT DUNGEON.DAT MANIFEST.txt" >&2
    exit 2
fi

output=$1
start_prg=$2
start_pak=$3
graphics_dat=$4
dungeon_dat=$5
manifest=$6

for input in "$start_prg" "$start_pak" "$graphics_dat" "$dungeon_dat"; do
    if [ ! -f "$input" ]; then
        echo "找不到輸入：$input" >&2
        exit 1
    fi
done

case "$output" in
    *.st) ;;
    *) echo "輸出必須使用 .st 副檔名：$output" >&2; exit 1 ;;
esac

# mformat -f 720 建立 80 tracks × 2 sides × 9 sectors 的 GEMDOS/FAT12 raw image。
# 所有原版輸入與輸出磁片都只放在呼叫端指定的私人工作目錄，不進 repo。
mformat -i "$output" -C -f 720 -v TALOSDM ::
mmd -i "$output" ::/AUTO
mcopy -i "$output" -o "$start_prg" ::/AUTO/START.PRG
mcopy -i "$output" -o "$start_pak" ::/START.PAK
mcopy -i "$output" -o "$graphics_dat" ::/GRAPHICS.DAT
mcopy -i "$output" -o "$dungeon_dat" ::/DUNGEON.DAT

{
    echo "Atari Talos 私人 GEMDOS 測試磁片收據"
    echo "輸出：$output"
    echo "注意：此收據只證明封裝輸入，不證明資料版本相容或遊戲可玩。"
    sha256sum "$start_prg" "$start_pak" "$graphics_dat" "$dungeon_dat" "$output"
    mdir -i "$output" -/ ::
} > "$manifest"

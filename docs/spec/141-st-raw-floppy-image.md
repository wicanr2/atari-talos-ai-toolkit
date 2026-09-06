# 141 — Atari ST raw 軟碟映像輸入契約

狀態：**READY**

## 範圍

本切片只定義 Talos 如何接收、驗證及唯讀保存未壓縮 `.st` 軟碟映像，並把
track／side／sector 轉成映像內的 512-byte sector。WD1772 command、DMA 傳輸、
DRQ／IRQ 時序、寫盤、MSA、STX、IPF 與 copy protection 不在本切片。

## 證據

- **已確認（Hatari 官方手冊）**：`.st` raw image 是實體磁片逐 sector 排列的映像；
  protected／非標準低階配置不能由 `.st` 完整保存，需 STX／IPF 等格式。
  <https://www.hatari-emu.org/doc/manual.html#Floppy-disk-images>
- **已確認（Atari ST 磁片格式文件）**：GEMDOS 磁片以 boot sector 的 BPB 描述
  bytes per sector、total sectors、sectors per track 與 sides；多位元組 BPB 欄位為
  little-endian。<https://disktype.sourceforge.net/doc/ch03s03.html>
- **已確認（專案權利與安全邊界）**：TOS ROM、遊戲磁片與原版素材由使用者自備，
  不加入 Git；映像輸入必須視為不可變，測試不能改寫來源。

## 契約

1. 輸入必須至少包含一個 512-byte boot sector。
2. BPB bytes/sector（offset `$0B`）目前只接受 512；total sectors（`$13`）、
   sectors/track（`$18`）與 sides（`$1A`）必須非零，sides 只接受 1 或 2。
3. `total sectors × bytes/sector` 必須與輸入長度完全相同；尾端垃圾、截斷及溢位
   一律失敗即關閉。
4. total sectors 必須可被 `sectors/track × sides` 整除；得到的 track 數必須非零。
5. sector number 採 WD1772／磁片慣例的 1-based；track、side 採 0-based。
   raw offset 為 `((track × sides + side) × sectors/track + sector - 1) × 512`。
6. 掛載時複製完整輸入；呼叫端之後修改原 slice 不得改變已掛載內容。
7. 本切片不宣稱任意 `.st` 都是 GEMDOS 磁片，也不支援靠低階異常軌保存的保護資訊。

## 驗收

- 合成 80-track、2-side、9-sector 映像能讀到第一、換面、換軌及最後 sector。
- sector 0、超界 track／side／sector 全部拒絕。
- 短 boot sector、非 512-byte sector、零幾何、非法 sides、長度不符及不可整除
  全部拒絕。
- 掛載後修改來源 slice，讀回資料仍不變。


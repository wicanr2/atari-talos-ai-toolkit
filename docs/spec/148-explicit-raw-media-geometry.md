# 148 — 明確指定 raw 媒體幾何

狀態：**READY**

## 證據與審查

- 已確認：既有規格 `141-st-raw-floppy-image.md` 定義逐 sector 的 raw 排列，
  公式為 `((track*sides+side)*sectorsPerTrack+sector-1)*512`。
- 已確認（輸入 bytes）：Automation 097 為 839680 bytes，BPB 宣告 819200 bytes。
  SHA-256 `afab0a6d2b41eec7420a76e174e96a906687cb959ea563f9343672f0b3febc23`。
  以 BPB 的 2 sides、10 sectors/track 計算，完整映像可容納 82 tracks。
- 未解：尾端兩軌的歷史用途及遊戲精確版本。長度算式不能證明原始實體磁片。
- 已確認（格式工具作者文件）：`.st` 是逐 sector 映像，GEMDOS 為 FAT12/16。
  <https://disktype.sourceforge.net/doc/ch03s03.html>。

## 契約

1. 保留既有 `NewRawFloppy` 的 BPB 精確長度契約。
2. 增加明確指定 tracks/sides/sectors 的建立入口，用於已知幾何的 raw 媒體。
   不讀改 BPB，不裁切、不補齊、不自動猜測。
3. tracks 與 sectors 非零、sides 為 1 或 2。乘法以 uint64 計算，
   完整檔案長度必須恰等於 `tracks*sides*sectors*512`。
4. 其餘不可變複本、CHS 邊界及原子掛載遵循規格 141。
5. 呼叫端必須在收據記錄指定的幾何與來源。指定幾何的實驗不能宣稱
   已證明磁片原始配置，也不能單獨證明遊戲啟動。

## 驗收

同一 82 軌映像的 BPB 宣告 80 軌時，預設入口拒絕，明確 82 軌入口接受；
驗證最後 sector 可回讀原 bytes、無別名、錯誤長度及零／非法幾何拒絕。

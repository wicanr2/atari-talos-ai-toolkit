# 144 — ST floppy A 槽面選擇

狀態：**CONFORMED**

## 範圍與停止線

本切片承接規格 143，讓已掛載的 raw `.st` A 槽依 PSG port A bit 0 選擇
side 0／1，再把該 side 帶入既有 Type-II 單 sector DMA。範圍不含 B 槽、寫盤、
WD1772 multiple-record、旋轉延遲與 copy-protected track；換軌只保留現有已觀察狀態，
另立規格。

## 證據

- **已確認（固定 EmuTOS 原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/floppy.c` 的
  `convert_drive_and_side()` 從 `$07` 開始，drive A 清 bit 1，side 1 清 bit 0；
  因此保留其他 port A 線位後，A/side 0 是 `$25`，A/side 1 是 `$24`。
  檔案 SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定 Hatari 正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 與私有混合素材
  720 KiB GEMDOS 磁片的 `fdc,psg_write` trace，在 VBL 260 明記
  `io_porta_old=0x5 io_porta_new=0x4 side 0->1 drive 0->0`，下一筆 Type-II read
  是 drive 0／track 0／side 1／sector 6；其後 side 1 sector 8、9 亦成功。
  VBL 335 再由 `$05→$04`，依序讀 track 1／side 1／sector 1–9。
- **已確認（現有 Talos）**：`RawFloppy.Sector(track, side, sector)` 已依 BPB 驗證
  1 或 2 面並以 CHS 計算偏移；規格 143 的媒體狀態機目前固定傳 side 0，遇 `$24`
  會失敗即關閉。

私有混合素材磁片只證明這條 I/O 路徑與 Hatari 的行為順序，不證明 Dungeon Master
資料相容、可玩性或原版 parity；正式對拍仍需合法的原版 ST 磁片。

## typed 行為

1. 媒體狀態機接受 port A `$25`（drive A／side 0）與 `$24`（drive A／side 1）；
   bit 1 必須保持 low 以選 A 槽。B 槽與未選取狀態仍由既有 VBL 路徑處理，不得誤入
   A 槽媒體讀取。side 1 保持選取時，下一次 `set_psg_porta` 的 select／read／
   `$24→$24` data 共用前置也必須可重入。
2. 每次合法 A 槽 port 寫入都將 `Drive=0`、`Side=0|1`、原始 `DrivePort` 寫入
   current receipt；後續 sector command 必須把 receipt 的 side 傳給
   `RawFloppy.Sector`。
3. side 必須小於已掛載映像 BPB 的 sides；單面映像選 side 1、其他 port 組合或
   超界 CHS 都在送出 command 時原子失敗，不改 RAM、DMA、IRQ 或 FDC command。
4. cold reset 清除 pending transfer 與 side 收據；規格 142–143 的固定延遲、
   `DMA_OK`、IRQ／GPIP5、連續 sector 與無媒體 timeout 不變。

## 驗收

- synthetic 兩面映像對 side 0／1 填入不同資料，分別經正式狀態機讀入 RAM，驗證
  exact bytes 與 receipt side／port。
- 單面映像選 side 1 及非法 port 都原子失敗；side 0、連續 sector 與無媒體測試不回歸。
- 固定 EmuTOS 加上述私有 bootstrap 磁片跨過 `$24` gate，至少完成 Hatari 已觀察到的
  track 0／side 1 讀取，再以下一個未支援 gate 收斂範圍。
- `go test ./...`、`go vet ./...`、`go build ./...` 全部通過後才升 **CONFORMED**。

## CONFORMED 收據

- 2026-09-06：固定 Hatari trace SHA-256
  `7f96bb4f9b451bb56e3c96b64855f38f89c82b26fd432680ff4b24e0e02838a2`；
  `$05→$04` 後的 A 槽 side 1／track 0／sector 6 與後續跨面順序已保存。
- Talos 由相同 EmuTOS 與私有 bootstrap 磁片從 reset 自然完成第 4 筆 receipt：
  `Drive=0`、`Side=1`、`DrivePort=$24`、track 0／sector 6、512 bytes；command clock
  `107,761,014`、DMA complete `107,921,270`、dummy-seek status read
  `107,924,966`，未走 timeout／force-interrupt。該點揭露 side 1 的 `$24→$24`
  共用前置必須可重入；補齊後同一工作負載再完成 side 1 sector 8／9，以及回到
  side 0 的 sector 8。下一個 gate 前移至 1,530,617 instructions／1,902 interrupts／
  110,862,604 clocks 的 DMA control `$FF8606` word write，屬後續 track 選擇切片。
- 兩面 exact bytes、單面 side 1 原子拒絕與 `$24` 共用前置分派均有永久測試；完整
  `go test ./...`、`go vet ./...` 與 `go build ./...` 通過。

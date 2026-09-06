# 142 — ST floppy 唯讀 sector DMA 完成路徑

狀態：**CONFORMED**

## 範圍與停止線

本切片把規格 141 掛載的 raw `.st` 媒體接到規格 123 已有的 WD1772 single-sector
read `$80`：drive A、side 0、track 0、sector 1 經 DMA 寫入 RAM，完成時產生 FDC IRQ，
由 EmuTOS 讀取 Type-II status `$80` 後接回既有 dummy-seek 收尾。多 sector、換面／換軌、
寫盤、CRC／RNF 錯誤、DRQ 逐 byte 精確時序與旋轉位置模型另立規格。

## 證據

- **已確認（規格 141）**：raw `.st` 依 BPB 驗證幾何，`RawFloppy.Sector` 以
  track／side 0-based、sector 1-based 回傳不可變的 512-byte sector。
- **已確認（固定 EmuTOS 原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1120–1165 flopio`
  逐 sector 設定 FDC sector、DMA address、sector count 1，再送 `FDC_READ`；
  `floppy.c:1544–1567 flopcmd` 等 GPIP5 active，之後讀 command/status register。
  檔案 SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定 Hatari 正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`、合成 720 KiB
  raw `.st` SHA-256
  `20047d4cf58ff096e5b4958513b4882ee51a465a371044c6f13174bb91c9fc10`；280-VBL FDC
  trace SHA-256 `f6e87e12f4ac730df11e7d97cffe7c8b1f6fa7af2cc44c62a0efe5ba04d844fa`。
  VBL235 送出 drive 0／side 0／track 0／sector 1、DMA `$001004`、count 1、command
  `$80`；VBL240 開始傳輸，DMA 位址每 16 bytes 前進，VBL241 抵達 `$001204` 後完成、
  拉低 IRQ。EmuTOS 寫 DMA control `$0090`、讀 DMA status 的 `DMA_OK`，再寫 `$0080`
  並讀得 Type-II FDC status `$80`，然後照常送 data 0／seek `$13`。

## typed 行為

1. 只有 drive A 已掛載、目前交易為 drive 0／track 0／sector 1、DMA count 1、DMA
   `$001004..$001203` 全在 RAM 內時，`$80` 才進成功讀取排程；不合條件失敗即關閉且
   不修改 RAM。未掛載媒體仍走規格 124 的 guest 1.5 秒 timeout。
2. 成功讀取排程採固定 `160,256` ST master clocks（一次 50 Hz frame）作為**可重現
   功能近似**。Hatari 證明首次完成約跨 5–6 VBL、後續讀取依旋轉位置不同；本切片不宣稱
   逐週期或逐 DRQ parity。玩家可見對拍只在載入完成後的穩定狀態取樣。
3. 到期時原子地把 512 bytes 寫入 DMA 起始位址、位址加 512、sector count 減為 0；
   FDC status 由 busy `$81` 成為 Type-II `$80`，IRQ active 並把 MFP GPIP5 拉低。
4. IRQ 後只接受 DMA control `$0090`、讀 `$FF8606` 得 `DMA_OK=$0001`、再寫 `$0080`
   並讀 `$FF8604`。FDC 讀取回傳 `$0080`、
   清 IRQ／抬高 GPIP5，接回既有 `floppyMediaSeekDataSelector`；dummy seek 完成後才把
   本輪 receipt 寫入有界 ring。
5. cold reset 取消 pending read 與排程，但不彈出已掛載媒體。錯 register、值、寬度、
   權限、順序、CHS 或 DMA 越界均失敗即關閉。

## 驗收

- 合成 sector 具唯一 pattern；到期前 RAM／DMA address／count 不變，到期後 512 bytes、
  address `$001204`、count 0、status／IRQ／GPIP 全部一致。
- 驗證 `$0090→DMA_OK read→$0080→FDC status read`，並證明其後可接回既有 dummy seek。
- 驗證未掛載仍走 timeout、錯序／DMA 越界不改 RAM、cold reset 清 pending。
- `go test ./...`、`go vet ./...` 與 `go build ./...` 通過後才升 **CONFORMED**。

## CONFORMED 收據

- 2026-09-06：固定 EmuTOS 1.3 UK ROM 由 reset 自然完成第一輪掛載媒體讀取；read
  command clock `106,340,810`、固定近似完成 clock `106,501,066`，dummy seek 在
  `106,504,762` 讀 status 收尾。本輪沒有 timeout selector 或 force interrupt，完成點為
  1,300,992 instructions／1,766 interrupts／106,504,776 clocks。
- 合成 sector 的 512 bytes 與 DMA RAM `$001004..$001203` 完全相同，DMA address 前進至
  `$001204`、count 歸零；`DMA_OK=$0001`、FDC status `$0080`、IRQ／GPIP5 read-clear
  與既有 dummy seek 均由測試覆蓋。DMA 越界會在 command 提交時原子失敗。
- 固定 ROM 的掛載媒體正常路徑、未掛載媒體舊回歸、完整 `go test ./...`、`go vet ./...`
  與 `go build ./...` 全部通過。

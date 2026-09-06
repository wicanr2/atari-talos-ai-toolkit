# 143 — ST floppy 同軌連續 sector 讀取

狀態：**CONFORMED**

## 範圍與停止線

本切片承接規格 142 的 single-sector 成功路徑，使同一次 EmuTOS `flopio()` 能在
Type-II status 成功後設定下一個 sector 與 DMA address，重複一個 sector 一個 command
的讀取，直到客體選擇 data register 並執行既有 dummy seek。範圍限目前已鎖定的 drive A、
side 0、track 0；換軌、換面、Type-II multiple-record bit、錯誤重試與寫盤另立規格。

## 證據

- **已確認（固定 EmuTOS 原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1081–1185 flopio`
  在 `while(count--)` 中逐 sector 重新寫 FDC sector、DMA address、DMA count 1 與
  `FDC_READ`；成功後 `userbuf += 512`、`sect++`。只有離開整個迴圈後才呼叫
  `flopunlk()`／`dummy_seek()`。檔案 SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定 Hatari 正常路徑）**：規格 142 同一份 Hatari 2.4.1／EmuTOS 1.3／
  合成 720 KiB raw `.st`，FDC trace SHA-256
  `f6e87e12f4ac730df11e7d97cffe7c8b1f6fa7af2cc44c62a0efe5ba04d844fa`。
  第二組讀取由 sector 1／DMA `$001004` 開始；每次依序完成 `DMA_OK` 與 FDC status
  `$80`，隨即以 `$0084` 設 sector 2／3／4…，DMA address 為 `$001204`／`$001404`／
  `$001604`…。每筆仍是 count 1、command `$80`，不是 WD1772 multiple-record command。
- **已確認（現有 Talos）**：規格 142 已驗證一筆 512-byte DMA、`DMA_OK`、FDC status、
  IRQ／GPIP5 與單 sector 後直接 dummy seek；未掛載媒體的 timeout 循環不可回歸。

## typed 行為

1. 成功讀取 FDC status 後進入明確 post-read phase。此 phase 只接受 `$FF8606=$0084`
   開始下一個 sector，或 `$FF8606=$0086` 進既有 dummy seek；其他交易失敗即關閉。
2. 下一 sector 接受 1-based 16-bit 值，但提交 command 時必須可由已掛載 raw image 的
   CHS 解析；不截斷、不回繞。DMA low／middle／high 仍須依既有順序寫入，但值由客體決定，
   完成的 22-bit even address 必須容納完整 512-byte RAM 範圍。
3. 每個 sector 都重做 `$0190→$0090→count 1→$0080/$0080`，沿用規格 142 的固定
   160,256-clock 功能近似、原子 DMA、IRQ、`DMA_OK` 與 Type-II status read-clear。
4. current receipt 保存最後一個 sector，並累計本次 `flopio()` 已完成的 sector 數與 bytes；
   只有 dummy seek status read-clear 後才進有界 ring。cold reset 清累計與 pending transfer。
5. 沒有媒體仍由第一筆 command 走原有 timeout／force-interrupt；本切片不得讓無磁片
   正常路徑誤入 post-read。

## 驗收

- synthetic 測試連續讀 sector 1、2、3 至三個不同 DMA buffer，逐筆驗證到期前不變、
  到期後 exact bytes、address/count/status/IRQ，最後接 dummy seek。
- sector 0／超界、DMA 三 byte 錯序與 RAM 越界均原子失敗。
- 固定 EmuTOS 從 reset 至少完成一個多 sector `flopio()`，資料與 raw image 對應範圍一致，
  receipt 的 sector／byte 累計正確，且沒有 timeout／force-interrupt。
- 未掛載固定 ROM 回歸、`go test ./...`、`go vet ./...`、`go build ./...` 全部通過後
  才升 **CONFORMED**。

## CONFORMED 收據

- 2026-09-06：固定 EmuTOS 從 reset 完成第一筆 single-sector `flopio()` 後，自然進入
  第二筆 6-sector `flopio()`。第二筆依序讀 sector 1–6，最後完成 clock
  `107,499,042`，dummy seek status read clock `107,502,734`；receipt 為 6 sectors／
  3,072 bytes，沒有 timeout selector 或 force interrupt。
- DMA RAM `$001004..$001C03` 與合成 raw image 的前 3,072 bytes 逐 byte 相同。sector 0、
  sector 10（9-sector geometry 越界）、DMA overflow 與三 byte 錯序均有失敗即關閉測試。
- 完成點為 1,391,231 instructions／1,797 interrupts／107,502,748 clocks。固定 ROM
  有／無媒體路徑、完整 `go test ./...`、`go vet ./...` 與 `go build ./...` 全部通過。

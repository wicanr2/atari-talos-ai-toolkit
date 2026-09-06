# 145 — ST floppy 跨 track seek 與後續 sector 讀取

狀態：**CONFORMED**

## 範圍與停止線

本切片承接規格 144，讓已掛載 raw `.st` 的 A 槽在 `flopio()` 指定 track 與目前
head track 不同時，先完成 WD1772 Type-I seek，再從該 track 進入既有 Type-II
sector DMA。範圍限 command `$13`、drive A、相同已選 side 與合法 BPB track；不含
restore-on-error、seek verify、write、B 槽、旋轉位置或逐週期 WD1772 模型。

## 證據

- **已確認（固定 EmuTOS 原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/floppy.c:1081–1185 flopio()`
  依序呼叫 `select(dev,side)`、`set_track(track)`，成功後才在 sector loop 設定
  `FDC_SR`／DMA／`FDC_READ`。`bios/floppy.c:1520–1535 set_track()` 在目標等於
  `fi->cur_track` 時不動作；不同時以 `set_fdc_reg(FDC_DR,track)` 寫 data register，
  再送 `FDC_SEEK | fi->actual_rate`，成功才更新 `fi->cur_track`。檔案 SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定 Hatari 正常路徑）**：Hatari 2.4.1／EmuTOS 1.3／私人 720 KiB
  bootstrap trace SHA-256
  `7f96bb4f9b451bb56e3c96b64855f38f89c82b26fd432680ff4b24e0e02838a2`。
  VBL 306 先選 A/side 0，依序寫 `$FF8606=$0086`、data 1、`$0080`、command
  `$13`；trace 明記 `dest_track=0x1`、head `0x0→0x1`。IRQ 後直接寫 `$0084`、
  sector 1，接著由 track 1／side 0 讀取。
- **已確認（現有 Talos）**：規格 125 的 same-track dummy seek 已有 `$0086→data→
  $0080→$13`、pending deadline、IRQ／GPIP5 與 status read；規格 141 的 raw image
  已能依 BPB 拒絕超界 track。規格 144 工作負載目前停在上述 `$0086`，phase idle。

私人混合素材磁片只用於暴露模擬器 I/O 缺口，不證明遊戲資料相容、可玩或原版 parity。

## typed 行為

1. shared PSG prefix 已選 drive A／side 0 或 1、media phase idle 且 track 已鎖定時，
   `$FF8606=$0086` 開始 pre-read track seek，建立 current receipt 並保存 Drive、Side、
   DrivePort 與 drive-write clock。
2. 下一筆 FDC data 必須是 BPB track 範圍內的 16-bit 值且可無損表示為 byte；保存為
   receipt Track／SeekData。之後只接受 `$0080` 與 command `$13`，其他順序或值均
   失敗即關閉。
3. seek command 拉高 GPIP5、建立 Type-I busy status 與 pending deadline；deadline
   採 3 ms/track 加既有 command completion 的硬體規格近似。這是
   hardware-spec approximation，不宣稱旋轉、step pulse 或 Hatari 逐 clock parity。
4. 到期後提交 head track、拉低 GPIP5 並設 IRQ。此 pre-read seek 不讀 status、
   不結束 receipt；客體以 `$0084` 直接選 sector register，後續沿用規格 142–144 的
   sector、DMA、status 與 dummy seek。收尾 dummy seek 的 data 必須等於已提交的
   head track，不再寫死 track 0。
   後續沒有再次 seek 的同軌 receipt 也必須由 head track 初始化，不得退回 track 0。
5. cold reset 將 head track 歸零並清 pending track seek。非法／超界 track 不改 RAM、
   FDC command、IRQ、head track 或 receipt phase。

## 驗收

- synthetic 兩面映像由 head track 0 seek 至 track 1，驗證到期前 head 不變、到期後
  IRQ／GPIP5、再讀 track 1 exact sector bytes 與 receipt Track／Side。
- 超界 track、錯序、錯 command 原子失敗；same-track dummy seek、side 0／1、連續
  sector 與無媒體 timeout 不回歸。
- 固定 EmuTOS 加私人 bootstrap 從 reset 至少完成 track 1／side 0／sector 1，並以
  下一個未支援 gate 收斂範圍。
- `go test ./...`、`go vet ./...`、`go build ./...` 全部通過後才升 **CONFORMED**。

## CONFORMED 收據

- 2026-09-06：固定 EmuTOS 加私人 bootstrap 從 reset 完成第一次 track 0→1 seek；
  seek data／command clocks 為 `110,863,778`，head 到期後提交為 1，接著完成
  track 1／side 0／sector 1 的 DMA：read command `110,890,738`、complete
  `111,050,994`。該次揭露 dummy seek data 也必須是目前 track 1，已由測試釘住。
- 修正後同一探針跑滿 8,000,000 steps 無未支援 gate，抵達 7,996,009 instructions／
  3,991 interrupts／177,899,648 clocks，head 已到 track 40。ring receipt 保留
  track 39／side 1／sector 6–9 與 track 40／side 0／sector 1–3；所有未經新 seek 的
  同軌 receipt 也由 `fdcHeadTrack` 初始化，不再誤讀 track 0。
- synthetic 驗證 deadline 前不提交、到期後 IRQ／GPIP5、track 1 exact sector bytes、
  track-aware dummy seek 與超界原子拒絕；完整 `go test ./...`、`go vet ./...`、
  `go build ./...` 通過。

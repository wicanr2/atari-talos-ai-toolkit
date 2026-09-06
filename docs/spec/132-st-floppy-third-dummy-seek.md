# 132 — ST floppy第三次timeout後dummy seek

狀態：**CONFORMED**。

## 範圍與停止線

承接規格131完成的第三次`$D0` force-interrupt，處理`flopunlk()`對track 0送出的
第三次Type-I dummy seek：data 0、seek `$13`、728-FDC-clock scheduler、GPIP輪詢及
status `$E4` read-clear。第三組收據獨立保存；其後`flopio()`錯誤回傳另立規格。

## 證據

- **已確認（EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/floppy.c:1322–1341 flopunlk`
  呼叫`1481–1501 dummy_seek`，依序寫track 0至data register、送
  `FDC_SEEK|actual_rate`=`$13`、等待IRQ並讀status；檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari／既有硬體切片）**：Hatari 2.4.1 oracle image digest
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`；
  400-VBL PSG＋FDC trace SHA-256
  `97a7a5f348aec08ff36b2c5d23973f043818a652ac7d6c2039c993ce372a1d08`與前兩輪
  已確認相同`$0086/$0000/$0080/$0013`、IRQ、`$0080` status `$E4` read-clear。
  同track／verify-off時序重用已驗證的728 FDC clocks，不新造週期。
- **已確認（固定Talos入口）**：EmuTOS官方1.3 192K UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。規格131後
  stage 61；下一gate於4,600,513 instructions／2,903 interrupts／142,980,490 clocks
  寫`$FF8606=$0086`，PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. stage 61依序只接受`$FF8606=$0086`、`$FF8604=$0000`、`$FF8606=$0080`及
   `$FF8604=$0013`，進stage 65並保存第三組data／seek command／start clock。
2. `$13`設Type-I busy／motor status `$E5`、IRQ inactive及GPIP5 high，啟動既有
   728-FDC-clock scheduler；完成後status `$E4`、IRQ active、GPIP5 low並進stage 66。
3. 等待期保存第三組inactive GPIP poll及IRQ-observed；stage 66只接受
   `$FF8606=$0080`，stage 67只在IRQ active時接受`$FF8604` word read，回`$00E4`、
   清IRQ、GPIP5 high、保存read clock並完成stage 68。
4. word access沿用bus-slot wait加4 device clocks；錯值、錯序、錯寬度、user access、
   過早或重複read皆失敗即關閉。cold reset清第三組收據，不修改前兩組或DMA buffer。

## 驗收與邊界

- synthetic覆蓋完整transaction、scheduler、poll／IRQ／read-clear、獨立收據、錯序、
  reset、早先收據及RAM不變。
- 固定ROM須自然完成stage 68，鎖定CPU／clock／receipt及跨裝置下一gate。
- 本切片不修改Dungeon Master規則、素材、存檔或權利邊界；固定ROM、完整240,000筆
  CPU corpus、全測試、`go vet -stdmethods=false ./...`與`go build ./...`通過才升
  **CONFORMED**。

## 驗收收據

- 固定ROM送出第三次dummy seek `$13`，start clock為142,981,658；九次inactive
  GPIP poll後觀察IRQ，status read clock為142,982,974。完成點為4,600,755
  instructions／2,903 interrupts／142,982,988 clocks。
- 完成時stage 68、FDC command `$13`、status `$E4`、Type-I、IRQ inactive、GPIP `$B1`；
  CPU在PC `$FC38A0`、prefetch `$4E75,$2F0A`。前兩輪收據與DMA buffer保持不變。
- 真正下一gate為4,601,570 instructions／2,903 interrupts／142,994,602 clocks的
  YM2149 `$FF8800` byte write；其語意留待後續規格，不把第三次dummy seek誤當最終返回。
- synthetic、固定ROM、完整240,000筆CPU corpus、全測試、vet與build均通過，本規格
  升為 **CONFORMED**。

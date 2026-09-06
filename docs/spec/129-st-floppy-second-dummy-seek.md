# 129 — ST floppy第二次timeout後dummy seek

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格128完成的第二次`$D0` force-interrupt。EmuTOS再次經`flopunlk()`
對track 0送Type-I dummy seek，等待IRQ後讀status清IRQ。本切片涵蓋data register 0、
seek `$13`、既有728-FDC-clock期限、GPIP輪詢與status `$E4`，並保存第二組獨立收據。
後續PSG transaction、額外status檢查、第三次retry及最終錯誤回傳另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1322–1341 flopunlk`
  呼叫`floppy.c:1481–1501 dummy_seek`，依序寫目前track 0至FDC data register、送
  `FDC_SEEK|actual_rate`=`$13`，等待IRQ後讀command/status register清IRQ。檔案
  SHA-256 `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（既有固定hardware切片）**：規格107與125已用相同固定Hatari來源及ROM
  證實same-track、verify-off seek使用Talos既有`fdcSeekDeadline()`的728 FDC clocks，
  第一次dummy seek自然得到九次inactive GPIP poll。本切片重用同一scheduler，不另造時序。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS官方1.3 192K UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；400-VBL
  PSG＋FDC trace SHA-256
  `97a7a5f348aec08ff36b2c5d23973f043818a652ac7d6c2039c993ce372a1d08`。
  VBL385在第二次`$D0`後依序記錄`$0086/$0000/$0080/$0013`；command complete後
  設IRQ，再寫`$0080`並讀status `$E4`清IRQ。其後首先出現YM2149 R14 transaction，
  因此不得由FDC-only序列直接跳到第三次sector read。
- **已確認（固定Talos入口）**：規格128完成後3,457,037 instructions／2,511
  interrupts／130,386,416 clocks為read stage 39、FDC command `$D0`；下一gate在
  3,457,115 instructions／130,387,154 clocks寫`$FF8606=$0086`，
  PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. stage 39只接受supervisor word write`$FF8606=$0086`並進stage 40；stage 40只接受
   `$FF8604=$0000`，保存第二組dummy-seek data 0並進stage 41。
2. stage 41只接受`$FF8606=$0080`並進stage 42；stage 42只接受`$FF8604=$0013`，
   保存第二組seek command／start clock，設Type-I busy／motor status `$E5`、清IRQ、
   GPIP5 inactive high並啟動既有seek scheduler。
3. 728-FDC-clock期限完成時清pending、status成`$E4`、設IRQ／GPIP5 active low並進
   stage 44；等待期間保存第二組inactive GPIP poll與IRQ-observed收據。
4. stage 44只接受`$FF8606=$0080`並進stage 45；其後只在IRQ active時接受
   `$FF8604` word read，回`$00E4`、清IRQ／恢復GPIP5 high、保存第二組read clock並
   完成stage 46。Type-I status型別保持成立。
5. 六筆DMA／FDC word access沿用shared-bus alignment並各增加4 device wait clocks。
   錯register、值、寬度、user access、錯序、過早或重複status read均失敗即關閉。
6. cold reset清第二組data／command／clock／poll／IRQ-observed／status-read收據；不得
   覆寫第一次dummy-seek收據或修改DMA buffer `$001004..$001203`。

## 垂直鏈、驗收與權利邊界

- 本切片只解鎖EmuTOS正常無磁片錯誤收尾，不修改Dungeon Master規則、資料、素材、
  畫面、存檔或發行權利。
- synthetic測試覆蓋完整transaction、既有seek scheduler、poll／IRQ／status read-clear、
  獨立收據、錯序／錯值、reset、第一次收據不變與DMA buffer不變。
- 固定ROM必須自然完成第二次dummy seek與status read，鎖定完整CPU／clock／receipt，
  再定位跨裝置真正最早的下一個typed gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然送出第二次dummy seek `$13`，command start clock為130,388,322；
  九次inactive GPIP poll後觀察IRQ，status read clock為130,389,638。完成點為
  3,457,357 instructions／2,511 interrupts／130,389,652 clocks。
- 完成時read stage 46、FDC command `$13`、status `$E4`、Type-I status、IRQ
  inactive、GPIP input `$B1`；第一次dummy-seek clocks 118,356,450／118,357,766
  與其poll／IRQ收據均未被覆寫。D/A、SSP=`$687C`、SR=`$2310`、PC=`$FC38A0`
  與prefetch=`$4E75,$2F0A`均鎖入固定ROM測試。
- 完成後既有模型正常處理中間status transaction；真正下一gate為3,516,206
  instructions／2,528 interrupts／130,971,490 clocks的YM2149 `$FF8800` supervisor
  byte write。此時R14仍為`$25`、媒體檢查count仍73，該PSG transaction屬第三次
  retry的drive 0同值重選，留給後續規格。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

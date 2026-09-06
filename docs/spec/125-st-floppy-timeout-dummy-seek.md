# 125 — ST floppy timeout後dummy seek

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格124已完成的`$D0` force-interrupt。EmuTOS由`flopio()`帶著
`EDRVNR`離開後呼叫`flopunlk()`；它必須對目前track 0送一筆Type-I dummy seek，等IRQ、
讀status清IRQ，才把FDC狀態轉成後續`flopvbl()`可讀的Type-I形式。本切片完整涵蓋
data register 0、seek `$13`、既有728-FDC-clock期限、GPIP輪詢與status `$E4`。
下一次sector 1讀取、第二次1.5秒timeout、drive deselect及最終錯誤回傳另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1158–1160 flopio`
  在`flopcmd(FDC_READ)`timeout後設`EDRVNR`且不重試，`floppy.c:1193`呼叫
  `flopunlk()`。`floppy.c:1322–1341 flopunlk`解釋並呼叫`dummy_seek()`；
  `floppy.c:1481–1501 dummy_seek`依序寫FDC data register為目前track 0、送
  `FDC_SEEK|actual_rate`=`$13`，最後讀command/status register清IRQ。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；330-VBL
  FDC trace SHA-256 `a0e4a318dfbe98d21788d1f56071827104beba085fedf7f130ba715bf19b2251`。
  VBL310在`$D0`後依序記錄`$0086`、data `$0000`、`$0080`、seek `$13`；無磁片
  motor start後command complete並設IRQ，再寫`$0080`、讀status `$E4`清IRQ。
  下一筆FDC transaction為sector selector `$0084`；但FDC-only trace無法排除其前的其他
  裝置存取，固定Talos因此另鎖真正最早的typed gate。
- **強證據（既有固定hardware切片）**：規格107已用相同固定Hatari來源與ROM證實
  same-track、verify-off seek在Talos使用`fdcSeekDeadline()`的728 FDC clocks換算，
  並以九次inactive GPIP poll後觀察IRQ；本切片重用同一scheduler，不另造時序。
- **已確認（固定Talos入口）**：規格124完成後，2,370,962 instructions／2,136
  interrupts／118,355,282 clocks抵達`$FF8606=$0086`；PC=`$FC3728`、
  prefetch=`$8606,$2039`，read stage 17且FDC command `$D0`。

## typed行為

1. read stage 17只接受supervisor word write`$FF8606=$0086`並進stage 18；stage 18
   只接受`$FF8604=$0000`，保存retry data register 0並進stage 19。
2. stage 19只接受`$FF8606=$0080`並進stage 20；stage 20只接受
   `$FF8604=$0013`，保存retry seek command／start clock，設Type-I busy／motor status
   `$E5`、清IRQ、GPIP5 inactive high並啟動既有seek scheduler。
3. 728-FDC-clock期限完成時，清pending、status成`$E4`、設IRQ／GPIP5 active low，
   並進stage 22。等待期間保存本輪inactive GPIP poll數，不能與開機探測收據混用。
4. stage 22只接受`$FF8606=$0080`並進stage 23；其後只在IRQ active時接受
   `$FF8604` word read，回`$00E4`、清IRQ／恢復GPIP5 high、保存read clock並完成
   stage 24。Type-I status型別保持成立。
5. 六筆DMA/FDC word handler沿用shared-bus alignment並各增加4 device wait clocks。
   錯register、值、寬度、user access、錯序、過早status read或重複read均失敗即關閉。
6. cold reset清本輪data／command／clock／poll／IRQ-observed／status-read收據；不得
   修改DMA buffer `$001004..$001203`。

## 垂直鏈、驗收與權利邊界

- 此切片只解鎖EmuTOS正常無磁片錯誤收尾，不修改Dungeon Master規則、資料、素材、
  畫面、存檔或發行權利。
- synthetic測試覆蓋完整transaction、既有seek scheduler、九次inactive poll、IRQ／
  status read-clear、錯序／錯值、reset與DMA buffer不變。
- 固定ROM必須自然完成dummy seek與status read，鎖定完整CPU／clock／receipt，再定位
  跨裝置的真正最早typed gate；不得只依FDC-only trace跳過中間存取。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然送出retry seek `$13`，command start clock為118,356,450；九次inactive
  GPIP poll後觀察IRQ，status read clock為118,357,766。完成點為2,371,204
  instructions／2,136 interrupts／118,357,780 clocks。
- 完成時read stage 24、FDC command `$13`、status `$E4`、Type-I status、IRQ inactive、
  GPIP input `$B1`；D/A、SSP=`$0F2A`、SR=`$2310`、PC=`$FC38A0`與
  prefetch=`$4E75,$2F0A`均鎖入固定ROM回歸測試。
- 真正最早的下一gate為2,371,983 instructions／118,369,110 clocks的YM2149
  `$FF8800` supervisor byte write；PC=`$FC36D0`、prefetch=`$000E,$1010`。這筆PSG
  transaction須先於FDC-only trace顯示的下一次`$0084` sector selector處理。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

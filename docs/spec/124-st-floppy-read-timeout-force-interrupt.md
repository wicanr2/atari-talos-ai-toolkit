# 124 — ST floppy讀取逾時後force-interrupt

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格123已提交且仍busy的WD1772 Type-II read-sector `$80`。固定無磁片
profile不會產生FDC IRQ；EmuTOS以自己的`hz_200`期限等待1.5秒，逾時後選回command
register並送出Type-IV force-interrupt `$D0`。本切片只完成這兩筆write及busy／IRQ
狀態收據；後續data-register／seek、重試、錯誤回傳與`flopunlk()`另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1544–1567 flopcmd`
  依`motor_on`選`MOTORON_TIMEOUT`或`MOTOROFF_TIMEOUT`，送出command後呼叫
  `timeout_gpip()`；逾時才執行`set_fdc_reg(FDC_CS,FDC_IRUPT)`與`fdc_delay()`。
  `floppy.c:156`定義motor-on期限為`3*CLOCKS_PER_SEC/2`，即300個`hz_200` tick／
  1.5秒。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c:3964–4004
  FDC_WriteCommandRegister`只允許Type-IV command中斷busy FDC；`fdc.c:3732–3767
  FDC_TypeIV_ForceInterrupt`在busy時保留Type-II status型別，依`$D0`低四位清IRQ，
  再完成command並清busy。檔案SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
- **強證據（固定Hatari正常路徑）**：oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；680-VBL
  FDC trace SHA-256 `9cb1d1ac50082934c7296b3b06fe53eada0ce9d8d20d9d9cea5bb4212924a9c0`。
  首筆`$80`於VBL235送出且保持busy；到VBL310才依序寫control `$0080`與command
  `$D0`。trace明記current `$80`被`$D0`中斷、IRQ clear及command complete；恰為
  75個50 Hz VBL，即1.5秒。
- **已確認（固定Talos入口）**：規格123完成點為1,286,164 instructions／1,761
  interrupts／106,340,824 clocks。保持IRQ inactive後自然等待，於2,370,837
  instructions／2,136 interrupts／118,354,080 clocks抵達`$FF8606=$0080` gate；
  PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. read stage 15、command `$80`、DMA mode `$0080`、Type-II status busy `$81`、IRQ
   inactive時，只接受supervisor word write`$FF8606=$0080`，保存timeout-selector
   clock並進stage 16；不得改DMA address、count或RAM。
2. stage 16只接受`$FF8604=$00D0`。保存force-interrupt command與clock，將FDC
   command設為`$D0`、清busy得到status `$80`，維持Type-II status型別、IRQ inactive與
   GPIP5 inactive high，完成stage 17。
3. 兩筆word handler沿用shared-bus alignment並各增加4 device wait clocks。期限由
   EmuTOS既有Timer C／`hz_200`正常路徑產生；Talos不另造第二個1.5秒scheduler。
4. 錯register、值、寬度、user access或錯序均失敗即關閉且原子不變。cold reset清
   timeout selector與force-interrupt收據。

## 垂直鏈、驗收與權利邊界

- 此切片只解鎖EmuTOS正常開機的無磁片錯誤路徑，不修改Dungeon Master規則、資料、
  素材、畫面、存檔或發行權利。
- synthetic測試覆蓋兩筆順序、錯序／錯值、4 wait clocks、Type-II status型別、IRQ／
  GPIP、DMA buffer不變與cold reset。
- 固定ROM必須自然等待既有1.5秒guest期限、完成`$0080/$D0`，鎖定CPU／clock／收據
  並定位下一個typed gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然沿既有Timer C／`hz_200`路徑等待：timeout selector write clock為
  118,354,092，force-interrupt `$D0` write clock為118,354,530；完成點是
  2,370,884 instructions／2,136 interrupts／118,354,544 clocks。
- 完成時read stage 17、FDC command `$D0`、status `$80`、Type-II status型別、IRQ
  inactive、GPIP input `$B1`；D/A、SSP=`$0F2A`、SR=`$2310`、PC=`$FC373A`與
  prefetch=`$4E75,$2F0A`均鎖入固定ROM回歸測試。
- 下一gate為2,370,962 instructions／118,355,282 clocks的`$FF8606=$0086`，即Hatari
  trace中的data-register selector；明確留給規格125。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

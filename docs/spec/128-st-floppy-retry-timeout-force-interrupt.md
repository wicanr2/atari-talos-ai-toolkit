# 128 — ST floppy retry逾時後force-interrupt

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格127仍busy的第二次WD1772 Type-II read-sector `$80`。固定無磁片
profile不會產生FDC IRQ；EmuTOS再次依自己的`hz_200`期限等待1.5秒，逾時後選回
command register並送Type-IV force-interrupt `$D0`。本切片保存獨立retry timeout
收據並更新busy／IRQ狀態；後續第二次dummy seek、第三次retry及最終錯誤回傳另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1544–1567 flopcmd`
  每次送command後依`motor_on`使用`MOTORON_TIMEOUT`並呼叫`timeout_gpip()`，逾時才
  `set_fdc_reg(FDC_CS,FDC_IRUPT)`。`floppy.c:156`定義期限為
  `3*CLOCKS_PER_SEC/2`，即300個`hz_200` tick／1.5秒。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c:3732–3767
  FDC_TypeIV_ForceInterrupt`與`fdc.c:3964–4004 FDC_WriteCommandRegister`證實busy
  Type-II被`$D0`中斷時保留status型別、依低四位清IRQ並完成command。檔案SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
- **強證據（固定Hatari正常路徑）**：oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS官方1.3 192K UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；400-VBL
  PSG＋FDC trace SHA-256
  `97a7a5f348aec08ff36b2c5d23973f043818a652ac7d6c2039c993ce372a1d08`。
  第二次`$80`在VBL310送出，VBL385才依序寫control `$0080`與command `$D0`；
  trace明記current `$80`被中斷、IRQ clear與command complete，正好75 VBL。下一筆
  transaction是`$0086` data-register selector。
- **已確認（固定Talos入口）**：規格127後2,372,203 instructions／2,136 interrupts／
  118,371,412 clocks為read stage 37、Type-II busy `$81`。真正下一gate在
  3,456,990 instructions／2,511 interrupts／130,385,952 clocks寫
  `$FF8606=$0080`，PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. stage 37、retry command `$80`、DMA mode `$0080`、Type-II busy `$81`且IRQ inactive
   時，只接受supervisor word write`$FF8606=$0080`，保存獨立retry timeout-selector
   clock並進stage 38；不得修改DMA address、count、RAM或第一次timeout收據。
2. stage 38只接受`$FF8604=$00D0`。保存retry force-interrupt與clock，將FDC command
   設為`$D0`、清busy得到status `$80`，維持Type-II status型別、IRQ inactive與GPIP5
   inactive high，完成stage 39。
3. 兩筆word access沿用shared-bus alignment並各增加4 device wait clocks；1.5秒期限
   由EmuTOS既有Timer C／`hz_200`正常路徑產生，Talos不另造scheduler。
4. 錯register、值、寬度、user access或錯序均失敗即關閉且原子不變。cold reset清
   retry timeout selector與force-interrupt收據。

## 垂直鏈、驗收與權利邊界

- 本切片只解鎖EmuTOS正常開機的第二次無磁片錯誤路徑，不修改Dungeon Master規則、
  資料、素材、畫面、存檔或發行權利。
- synthetic測試覆蓋兩筆順序、獨立收據、錯序／錯值、wait、Type-II status、IRQ／
  GPIP、DMA buffer不變、第一次收據不變與cold reset。
- 固定ROM必須自然等待第二次1.5秒guest期限、完成`$0080/$D0`，鎖定CPU／clock／
  收據並定位下一個typed gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然沿既有Timer C／`hz_200`路徑再等待75個50 Hz VBL：retry timeout
  selector write clock為130,385,964，force-interrupt `$D0` write clock為
  130,386,402；完成點是3,457,037 instructions／2,511 interrupts／130,386,416 clocks。
- 完成時read stage 39、FDC command `$D0`、status `$80`、Type-II status型別、IRQ
  inactive、GPIP input `$B1`；第一次timeout clocks 118,354,092／118,354,530未被
  覆寫。D/A、SSP=`$687C`、SR=`$2310`、PC=`$FC373A`與prefetch=`$4E75,$2F0A`
  均鎖入固定ROM測試。
- 下一gate為3,457,115 instructions／2,511 interrupts／130,387,154 clocks的
  `$FF8606=$0086`，即固定Hatari trace中的第二次dummy-seek data-register selector。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

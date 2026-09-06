# 131 — ST floppy第三次逾時後force-interrupt

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格130仍busy的第三次WD1772 Type-II read-sector `$80`。固定無磁片
profile不產生FDC IRQ；EmuTOS依同一個`hz_200`期限再等待1.5秒，逾時後選回command
register並送Type-IV force-interrupt `$D0`。本切片保存第三組timeout收據並更新
busy／IRQ狀態；後續第三次dummy seek與`flopio()`最終錯誤回傳另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1544–1567 flopcmd`
  依`motor_on`使用`MOTORON_TIMEOUT`並呼叫`timeout_gpip()`，逾時才
  `set_fdc_reg(FDC_CS,FDC_IRUPT)`。`floppy.c:156`定義期限為
  `3*CLOCKS_PER_SEC/2`，即300個`hz_200` tick／1.5秒；檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c:3732–3767
  FDC_TypeIV_ForceInterrupt`與`3964–4004 FDC_WriteCommandRegister`證實busy
  Type-II被`$D0`中斷時保留status型別、依低四位清IRQ並完成command；檔案
  SHA-256 `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS官方1.3 192K UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；400-VBL
  PSG＋FDC trace SHA-256
  `97a7a5f348aec08ff36b2c5d23973f043818a652ac7d6c2039c993ce372a1d08`。
  VBL389送第三次`$80`後，無磁片仍busy；依前兩輪已確認的75-VBL契約，第三輪
  同樣由`$0080/$D0`中斷。這是迴圈同一原始碼路徑，不外推成功傳輸。
- **已確認（固定Talos入口）**：規格130後3,516,426 instructions／2,528 interrupts／
  130,973,792 clocks為stage 59、Type-II busy `$81`。下一gate自然出現在
  4,600,388 instructions／2,903 interrupts／142,979,288 clocks的`$FF8606` word
  write；PC=`$FC3728`、prefetch=`$8606,$2039`，與前次timeout selector入口相同。

## typed行為

1. stage 59、第三次command `$80`、DMA mode `$0080`、Type-II busy `$81`且IRQ
   inactive時，只接受supervisor word write`$FF8606=$0080`，保存第三組
   timeout-selector clock並進stage 60；不得修改DMA、RAM或前兩組timeout收據。
2. stage 60只接受`$FF8604=$00D0`。保存第三組force-interrupt與clock，將FDC
   command設為`$D0`、status `$81→$80`，維持Type-II型別、IRQ inactive與GPIP5 high，
   完成stage 61。
3. 兩筆word access沿用shared-bus alignment並各增加4 device wait clocks；1.5秒期限
   由guest既有Timer C／`hz_200`正常路徑產生，Talos不另造scheduler。
4. 錯register、值、寬度、user access或錯序均失敗即關閉且原子不變；cold reset清
   第三組timeout收據。

## 垂直鏈、驗收與權利邊界

- 本切片只解鎖EmuTOS正常開機的第三次無磁片錯誤路徑，不修改Dungeon Master規則、
  資料、素材、畫面、存檔或發行權利。
- synthetic覆蓋兩筆順序、獨立收據、錯序／錯值、wait、Type-II狀態、IRQ／GPIP、
  DMA buffer不變、前兩組收據不變與cold reset。
- 固定ROM必須自然完成第三次`$0080/$D0`，鎖定CPU／clock／receipt與下一gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然沿既有Timer C／`hz_200`再等待75個50 Hz VBL；第三次timeout selector
  bus clock為142,979,300，force-interrupt `$D0` bus clock為142,979,738。完成點為
  4,600,435 instructions／2,903 interrupts／142,979,752 clocks。
- 完成時stage 61、FDC command `$D0`、status `$80`、Type-II型別、IRQ inactive、GPIP
  input `$B1`；前兩輪timeout收據與DMA buffer保持不變。CPU回到PC `$FC373A`、
  prefetch `$4E75,$2F0A`的同一函式邊界。
- 下一gate為4,600,513 instructions／2,903 interrupts／142,980,490 clocks的
  `$FF8606=$0086`，即第三次dummy-seek data-register selector。
- synthetic、固定ROM、完整240,000筆CPU corpus、全測試、
  `go vet -stdmethods=false ./...`與`go build ./...`均通過，本規格升 **CONFORMED**。

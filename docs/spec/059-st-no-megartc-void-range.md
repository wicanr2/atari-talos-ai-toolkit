# 059 — Atari ST 無 Mega-RTC 的 `$FFFC21–$FFFC3F` void byte range

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理首版 `MACHINE_ST` 沒有 Mega-ST RTC 時，`$FFFC21–$FFFC3F` 的 byte
read／write 行為，解開 EmuTOS 1.3 `detect_megartc()`。它不實作 RP5C15 RTC 或主機時間。

- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMem.c` SHA-256
  `ba67024fc35deeed202276e95844effbc014327f0893af71b85102601d2103cd`。
  初始化在 `MACHINE_ST`／`MACHINE_STE` 將完整 `$FFFC21–$FFFC3F` read／write
  改成 void handlers；void read 回 `$FF`，write 不提交裝置狀態。
- **已確認（固定 EmuTOS 1.3 原始碼）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/clock.c` SHA-256
  `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
  `detect_megartc()` 先對 `$FFFFFC21` 執行 `check_read_byte()`，再以 byte read／write
  測試 bank 1 的 alarm registers；普通 ST 的 void range 使寫入無法回讀，最終不會
  誤判成有 Mega-RTC。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM／ST 參數與規格 058 相同。
  `$FC0636` 的 `TST.B (A0)` 前 A0=`$FFFFFC21`、SR=`$2704`、
  prefetch=`$4A10,$4E71`、FrameCycles=35088；執行後到 `$FC0638` 為 35096、
  SR=`$2708`、prefetch=`$4E71,$7001`，確認 byte read `$FF`、8 clocks、無例外。

## typed 行為

1. supervisor data FC=5 對 `$FFFC21–$FFFC3F` 任一位址的 `ReadByte` 回 `$FF`。
2. supervisor data FC=5 對該範圍的 `WriteByte` 成功但丟棄值；後續 read 仍回 `$FF`。
3. 24-bit address masking保留；user-mode I/O protection仍優先並回 `FaultProtected`。
4. `$FFFC20/$FFFC40`、word／long access、Mega-ST RTC、STE／MegaSTE／Falcon 與
   主機 wall-clock 均不在本切片，維持既有失敗即關閉。

## 驗收與停止線

- memory 測試涵蓋 base／middle／end、24-bit alias、discarded write、user protection、
  兩側相鄰位址與 word access 未被意外放寬。
- 固定 EmuTOS 完成第 6,879 條 `$FFFC21` `TST.B`，對拍 8 clocks、完整 CPU state
  與 prefetch；再有界步進找下一停點。
- 通過完整 CPU 語料、固定 ROM、Go 測試、靜態檢查與建置後升 **CONFORMED**。

## CONFORMED 收據

- memory 測試通過 range base／middle／end、24-bit alias、read `$FF`、discarded byte
  write、user protection、相鄰 reserved I/O 與 word access 失敗即關閉。
- 固定 EmuTOS 第 6,879 條為 8 clocks；完成後 D/A、SSP=`$0F80`、SR=`$2708`、
  PC=`$FC063C`、prefetch=`$4E71,$7001` 對上 Hatari tracepoint。
- 後續成功執行到第 6,916 條；下一停點是第 6,917 條對 `$FF8A3C` 的 Blitter
  探測 bus fault，需由 `TST.B` vector 2 切片接手。

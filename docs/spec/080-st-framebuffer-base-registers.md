# 080 — ST framebuffer 程式化基址暫存器

狀態：**CONFORMED**。

## 範圍與證據

本切片只建立普通彩色 ST／STF `$FF8201`（高位元組）與 `$FF8203`（中位元組）的
supervisor byte read／write、22-bit DMA 遮罩、cold reset，以及固定 EmuTOS 設定
`$0F8000` 的兩筆指令。掃描 counter `$FF8205/$FF8207/$FF8209`、目前作用中的
`VideoBase`、下一幀重載事件、STE `$FF820D` 低位元組及 framebuffer 像素輸出均未涵蓋。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；hardware map
  定義 `$FF8201/$FF8203` 為 display memory base address 的 high／middle byte。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `ioMemTabST.c` 將兩者映射為 byte R/W；`Video_ScreenBase_WriteByte` 對高位元組套用
  `DMA_MaskAddressHigh()`，普通 ST／STE、RAM 不超過 4 MiB 時 mask 為 `$3F`；
  `Video_GetScreenBaseAddr` 在 ST 只組成 `(high<<16)|(middle<<8)`。寫入只更新程式化
  暫存器；`Video_RestartVideoCounter` 才在新 VBL 前三條 HBL 的 cycle 48 重載作用中
  `VideoBase`。本專案不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  `$FC67FA: 11 C1 82 01` 是 `MOVE.B D1,$8201`，執行前 CycleCounter=403,924、
  D1=`$0F`、兩暫存器與作用中 `VideoBase` 皆 0；12 clocks 後高位元組為 `$0F`，
  `VideoBase` 仍為 0。`$FC67FE: E088` 將 D0 `$000F8000` 右移 8 位，24 clocks 後
  D0=`$00000F80`；`$FC6800: 11 C0 82 03` 再以 12 clocks 寫中位元組 `$80`，
  程式化基址成為 `$0F8000`，作用中 `VideoBase` 仍為 0。

## typed 行為

1. `Memory` 分別保存高／中 byte；cold reset 清零。supervisor byte read回保存值，
   24-bit alias 正常；user、word／long、相鄰位址均沿用 I/O memory map 失敗即關閉。
2. `$FF8201` write保存 `value & $3F`；`$FF8203` 保存完整 byte。這是實體 ST 的
   22-bit Shifter DMA address 上限，不依本切片只提供 512 KiB／1 MiB RAM 而再縮窄。
3. `ProgrammedVideoBase()` 回 `(high<<16)|(middle<<8)`，因此必為 256-byte aligned。
   此 API 明稱 programmed；不得把它當成目前掃描中的 active base。
4. 兩暫存器沿用 Shifter CPU bus 4-clock slot alignment；固定兩筆 `MOVE.B Dn,abs.w`
   各為 12 clocks。寫入不產生 scheduler side effect，也不重載 active base。

## 驗收與停止線

- synthetic tests 覆蓋 reset、readback、高位遮罩、組址、alias、權限、寬度、相鄰位址、
  timed wait，並確認失敗不提交。
- 固定 ROM 必須在 `$FC67FA/$FC6800` 兩個 instruction boundary 對上完整 D/A、SSP、SR、
  PC、prefetch、clock 與 register 值；第三筆完成後 programmed base 必須是 `$0F8000`。
- 完整 corpus、ST tests、固定 ROM、`go vet -stdmethods=false ./...` 與 build 全綠後才升
  **CONFORMED**；再有界續跑至下一 typed gate。active base latch 另立規格，不以本切片
  冒稱 framebuffer 已能顯示。

## 091 — ST IKBD ACIA first transmit deadline

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 寫入第一個 IKBD command byte `$80`，以及下一個
1024 CPU clocks 的 serial timer deadline。第二個 byte `$01`、完整 8N1 shift、IKBD
firmware response、RX、IRQ 與 MIDI ACIA 均不在範圍。

- **已確認（MC6850 平台契約）**：CPU 寫 transmit data register 後 TDRE 清零；資料從
  TDR 移到 transmit shift register 後，TDRE 再置位。TDRE 表示 TDR 可再接受資料，不代表
  整個 serial byte 已送完。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：CR=`$96` 在 FrameCycles 129,988 建立
  divider=64、timer=1024 CPU clocks；TDR=`$80` 在 130,272 寫入，status 隨即由 `$02`
  變 `$00`；timer deadline 後，guest 在 131,088 讀回 `$02`。
- **已確認（固定 EmuTOS ROM）**：SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；正常路徑第一個
  data-register write 是 `$80`。

## typed 行為與驗收

1. configured 且 TDRE=`1` 時，唯一允許的首筆 data write 是 `$80`；存入 TDR、標記待移位，
   並清除 TDRE。其他值、錯序、重複與不支援寬度一律失敗即關閉且原子不變。
2. CR=`$96` 完成的指令邊界起，每 1024 clocks 產生 ACIA serial deadline。第一個 deadline
   若有待移位 TDR，清除 pending 並恢復 TDRE=`1`。
3. 目前排程以 MC68000 指令邊界作可重現近似；Hatari 從裝置寫入 phase 起算。這項 phase
   差異必須明載，不能宣稱逐 cycle parity。
4. synthetic memory/scheduler test、固定 ROM、完整 corpus、全測試、vet 與 build 通過，
   並抵達第二個 data byte 或下一 gate 後才升 **CONFORMED**。

## 驗收收據

- synthetic memory test 確認 `$80` 寫入會清 TDRE、第一 deadline 恢復 TDRE，且第二 byte
  仍維持範圍外的失敗即關閉。
- synthetic scheduler test 確認 deadline 前不提前更新、deadline 當下更新且下一期為
  `+1024` clocks。
- 固定 ROM 已抵達第二個 data byte `$01` gate：68,645 instructions、4 interrupts、
  969,640 clocks。
- 完整 corpus、全測試、`go vet` 與建置結果記錄於專案驗證矩陣。

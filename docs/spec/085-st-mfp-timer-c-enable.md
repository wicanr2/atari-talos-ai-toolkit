# 085 — ST MFP Timer C interrupt enable

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 路徑在 Timer C 已啟動、尚未 timeout 且 IPRB=`$00` 時，
把 IERB `$FFFA09` bit 5 從 0 設為 1。timeout、pending、mask 後 IRQ、priority、IACK
與其他 channel 不在範圍。

- **已確認（MC68901 一手規格）**：NXP user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`；
  IERB bit 5 對應 Timer C。寫 1 啟用 channel，寫 0 禁用並清該 channel pending；
  interrupt enable 與 mask 是兩級獨立 gate。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`mfpint()` 在安裝 vector 後以
  `mfp->ierb |= mask`、`mfp->imrb |= mask` 啟用 channel。Timer C 的 interrupt number
  是 5，故 mask=`$20`。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：在 Timer C `$50` 啟動後，trace
  依序顯示 IMRB=`$00`、IERB=`$00`、IPRB=`$DF`、ISRB=`$DF`，再於 FrameCycles
  124,638 寫 IERB=`$20`，FrameCycles 124,674 寫 IMRB=`$20`；中間沒有 Timer C
  timeout 或 MFP exception。

## typed 行為與驗收

1. IERB=`$00`、IPRB=`$00`、TCDCR Timer C control=`5` 時，唯一新增允許的 write
   是 `$20`；commit IERB bit 5，不憑空設定 pending。
2. Timer C 未啟動、IPRB 非零、其他 enable bits、重複 `$20` 或 mixed value 仍回
   `unsupported_device_state`，且 IERB／IPRB 原子不變。
3. reset、read、write-zero、4 wait clocks、alias、權限與寬度沿用規格 064。
4. synthetic test與固定 ROM已通過；IERB／IMRB 最終為 `$20/$20`。正常路徑前進到
   68,378 instructions、4 interrupts、966,808 clocks；下一 gate 是 Timer D 啟動的
   TCDCR `$50→$51`，完整 state 為 SSP=`$0F3E`、SR=`$2300`、pipeline PC=`$FC629E`、
   prefetch=`$001D,$1142`。完整 240,000 筆 CPU corpus、全測試、vet 與 build通過後，
   本規格升 **CONFORMED**。

本切片不改遊戲規則、畫面、存檔或權利邊界；它只延伸公開 EmuTOS 開機路徑。

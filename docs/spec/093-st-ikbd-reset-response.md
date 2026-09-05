# 093 — ST IKBD reset response

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 color-ST／EmuTOS 1.3 UK 路徑送完 IKBD warm-reset command
`$80,$01` 後的第一個 response `$F1`。一般鍵盤／滑鼠封包、其他 IKBD commands、RX
overrun、MFP interrupt routing 與 MIDI ACIA不在範圍。

- **強證據（固定 Hatari 2.4.1 外部 oracle）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。Hatari在
  VBL6/HBL293收完 `$01` 並啟動 reset timer；VBL10/HBL21完成 timer，HBL25開始送
  `$F1`，HBL43令 ACIA RDR=`$F1`，HBL59 guest讀 status=`$83`、再讀 RDR=`$F1`，
  read後 status=`$02`。從 command receive到 RDR full是 1,002 scanlines，固定50 Hz
  profile為 513,024 CPU/video clocks。
- **已確認（MC6850平台契約）**：status bit 0是 RDRF、bit 7是 IRQ；讀 RDR清 RDRF與
  receive IRQ，TDRE bit 1不受影響。CR=`$96`啟用 receive interrupt。
- **已確認（固定 Talos正常路徑）**：規格092在第二 TDR移入 shift stage後繼續輪詢
  `$FFFC00` 的 RDRF，沒有其他 bus gate。

## typed 行為與驗收

1. 第二個 `$01` frame的第10個 serial tick完成時，建立唯一 reset-response deadline：
   current machine clock +513,024。不得在 command尚未完整送出時排程。
2. deadline前 status維持 TDRE且無RDRF；跨 deadline時 RDR=`$F1`、status置 `$83`。
3. configured supervisor byte read `$FFFC02`在RDRF時回 `$F1`，原子清 bit 7與bit 0，
   status回 `$02`；提早讀、重讀、user或word access失敗即關閉。
4. 本切片只建模ACIA status IRQ bit，不宣稱MFP GPIP／vector interrupt已接線。513,024
   是固定 Hatari color-ST profile收據，不外推為所有IKBD ROM或顯示模式的通則。
5. synthetic deadline／RDR test、固定 ROM、完整 corpus、全測試、vet與build通過，
   並抵達下一個typed gate後才升 **CONFORMED**。

## 驗收收據

- synthetic test確認513,024-clock deadline前不提早置位，deadline時RDR=`$F1`、
  status=`$83`；讀取後status=`$02`，重讀失敗即關閉。
- 固定 ROM正常路徑在128,313 instructions、8 interrupts、1,507,268 clocks讀取 `$F1`；
  Talos裝置deadline為1,503,070 clocks；D0 low byte、D1 status、PC／prefetch、ACIA state
  與response deadline均固定回歸。
- 讀取後正常路徑繼續至136,048 instructions、8 interrupts、1,577,208 clocks，下一個
  typed gate是MIDI ACIA control `$FFFC04` 的reset write。
- 本切片沒有接MFP ACIA IRQ，也沒有產生一般鍵盤／滑鼠事件。
- 完整corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。

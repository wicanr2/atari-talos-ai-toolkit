# 092 — ST IKBD ACIA second transmit buffer

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 在首筆 IKBD command `$80` 已進入 transmit shift
register 後，把第二個 command byte `$01` 寫入 TDR，並等待前一個 8N1 frame 的 10 個
serial ticks 結束後再移位。IKBD command 語意、MCU response、RX、IRQ 與 MIDI ACIA
均不在範圍。

- **已確認（MC6850 平台契約）**：TDR 與 transmit shift register 分離；shift register
  busy 時可以先填下一個 TDR，但 TDRE 必須維持清除，直到該 TDR 真正移入 shift register。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：固定 EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；trace 在
  FrameCycles 131,160 寫 TDR=`$01`，131,304 status仍為 `$00`。`$80` 於 HBL 255
  prepare，HBL 257–271 傳 8 data bits、HBL 273 stop、HBL 275 才 prepare `$01`；
  guest下一次於 145,082 觀察 status=`$02`。
- **已確認（固定 Talos 正常路徑）**：規格 091 後在 68,645 instructions、4 interrupts、
  969,640 clocks 抵達 `$FFFC02` 的第二筆 data write，D2 low byte=`$01`。

## typed 行為與驗收

1. 首筆 `$80` 從 TDR 移入 shift register 時，建立 10 個 serial ticks 的 busy counter，
   同時恢復 TDRE。
2. shift busy、TDRE=`1`、前一筆 TDR=`$80` 時，唯一允許的第二筆 write 是 `$01`；更新
   TDR、標記 pending並清 TDRE。其他值、錯序與重複寫入皆失敗即關閉且原子不變。
3. 每個 1024-clock deadline遞減 busy counter；第 10 tick 將 `$01` 從 TDR 移入 shift
   register、清 pending、恢復 TDRE，並開始下一個 10-tick frame。
4. 這是 serial framing與buffer ownership的最小模型，不宣稱逐 bit輸出或 IKBD firmware
   已收到 command。Hatari只在稍後 guest poll觀察 TDRE；Talos不得把該 poll時間冒稱
   裝置內部 transfer的精確時刻。
5. synthetic buffer／deadline測試、固定 ROM、完整 corpus、全測試、vet與build通過，
   並抵達下一個 typed gate後才升 **CONFORMED**。

## 驗收收據

- synthetic buffer測試逐 tick確認前 9 ticks維持 TDRE=`0`，第 10 tick才移入 `$01`、
  清 pending並恢復 TDRE=`1`。
- 固定 ROM的第二次 ACIA transfer deadline為 979,806 clocks；CPU在跨越該 deadline後
  的 observation boundary可能因指令長度稍晚，測試固定裝置clock而不冒稱兩者相同。
- 下一個玩家路徑邊界是 `$FC48AA` 等待 RDRF；Hatari確認 IKBD warm-reset完成後回傳
  `$F1`，status=`$83`，guest讀取後 status回到 `$02`。RX／IRQ另立規格，不混入本切片。
- 完整 corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。

# 097 — ST MFP Timer D system-clock start

狀態：**CONFORMED**。

## 範圍與證據

本切片處理固定EmuTOS把早期USART Timer D設定（data 2、control 1）改成系統時鐘
（data 256、control 2），並啟用MFP channel 4。Timer D countdown／reload／timeout／
pending／IACK與CPU handler不在範圍。

- **已確認（MC68901一手規格）**：Timer D使用TCDCR bits2–0；control 2是delay mode
  prescaler ÷10。timer data byte `$00`代表256，非0。channel 4對應IERB/IPRB/ISRB/IMRB
  bit4。
- **強證據（固定Hatari 2.4.1外部oracle）**：固定EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。VBL10
  trace依序為IMRB=`$60`、IERB=`$60`、IPRB=`$EF`、ISRB=`$EF`、TCDCR `$51→$50`、
  TDDR `$02→$00`、IERB `$60→$70`、IMRB `$60→$70`、TCDCR `$50→$52`；最後
  Hatari記錄`data=256 ctrl=2 timer_cyc=2560 first=true`。
- **已確認（固定Talos正常路徑）**：規格096後在136,182 instructions、8 interrupts、
  1,578,882 clocks抵達IERB同值`$60`，D0=`$FFFFFFEF`。

## typed行為與驗收

1. ACIA channel stage完成、IERB/IMRB=`$60`時，以IERB同值`$60`建立本序列stage。
2. 依序以`$EF`對IPRB、ISRB作write-zero-to-clear；跳步或錯值失敗即關閉。
3. stage完成才允許TCDCR `$51→$50`停止Timer D、清running transition；Timer C高nibble
   保持control 5。停止後TDDR=`$00`同步寫data/main counter，typed語意為256。
4. 只允許IERB `$60→$70`、IMRB `$60→$70`後，才以TCDCR `$50→$52`重新啟動
   Timer D並latch start transition。未實作scheduler前不宣稱timeout或IRQ。
5. synthetic staged test、固定ROM、完整corpus、全測試、vet與build通過，並抵達
   下一個typed gate後才升 **CONFORMED**。

## 驗收收據

- synthetic測試確認八段stage、`$00` data latch、`$52` start與禁止跳過clear。
- 固定ROM在136,210 instructions、8 interrupts、1,579,228 clocks完成TCDCR=`$52`；
  完整CPU state／prefetch與IERB/IMRB/TCDCR/TDDR/main均固定回歸。
- 啟動後沒有新的I/O fault，而是等待尚未建模的Timer D timeout；下一規格入口是
  2560 MFP ticks recurrence、channel 4 pending與MFP interrupt acknowledge。
- 完整corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。

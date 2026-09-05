# 086 — ST MFP Timer D delay-mode start

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 開機路徑在 Timer C control=`5` 持續運作時，
把 Timer D 從停止狀態啟動為 delay mode、control=`1`（prescaler ÷4）的 TCDCR
`$FFFA1D` 寫入。Timer D countdown、reload、USART baud clock、timeout 與 interrupt
不在本切片；這些行為未接線前不得宣稱 serial timer 已完成。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §6.2.2。Timer D 使用 TCDCR bits 2–0；control `1` 是 delay mode ÷4，Timer C 與 D
  的 control 欄位可獨立改寫。
- **已確認（固定 EmuTOS 1.3 原始碼與 ROM）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`。Timer D 初始化呼叫 `xbtimer()`，
  `setup_timer()` 保留 Timer C 高 nibble、先停止 D、寫 TDDR=`$02`，再把 control `1`
  合併成 TCDCR=`$51`。固定 ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；失敗寫入位於
  `$FC62AA` 共用 helper。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`。trace 先在 FrameCycles 128,116
  寫 TCDCR=`$50`、128,132 寫 TDDR=`$02`，再於 128,176 寫 TCDCR=`$51`；
  隨後記錄 `data=2 ctrl=1 timer_cyc=8 first=true`。固定 ST clock 比例換算為
  `floor(8×8,021,248÷2,457,600)=26` CPU clocks。

## typed 行為

1. TCDCR=`$50` 時先寫 `$50` 是停止 Timer D 並保留運作中 Timer C 的同值操作；
   register／transition 不變。接著 TDDR/main=`$02` 時，寫 `$51` 保留 Timer C control 5，設定
   Timer D control 1，並 latch 一次 Timer D start transition。
2. write 不改 TCDR/main、IERB、IPRB、IMRB 或既有 Timer C start state。
3. 資料不符、Timer C control 不符、active Timer D 的重複／停止、其他 Timer D control 與 mixed
   values 都回 `unsupported_device_state`，register 與 transition 原子不變。
4. reset 清 Timer D transition；read、4 wait clocks、alias、權限與寬度契約不變。
5. transition 是 scheduler 的 typed 邊界，不等同 26-clock recurrence 已接線。

## 驗收與停止線

- synthetic test涵蓋 `$50→$51`、兩個 timer control 共存、錯誤 data／control、
  原子失敗、reset 與既有 I/O 邊界。
- 固定 EmuTOS 已依序完成 TCDCR `$50`、TDDR `$02`、TCDCR `$51`，並抵達
  68,392 instructions、4 interrupts、966,948 clocks；下一個 typed gate 是 UCR
  `$FFFA29` 寫 `$88`，完整 state 為 SSP=`$0F3E`、SR=`$2300`、pipeline PC=`$FC6B3E`、
  prefetch=`$FA29,$4A45`。完整 240,000 筆 CPU corpus、固定 ROM、全測試、vet 與
  build通過後，本規格升 **CONFORMED**。

本切片只延伸公開 EmuTOS 開機路徑，不改 Dungeon Master 規則、畫面、存檔或權利邊界。

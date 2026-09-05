# 090 — ST IKBD ACIA control initialization

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 對 IKBD MC6850 ACIA control/status `$FFFC00` 的
master reset `$03`、configuration `$96` 與 configuration 後 transmitter-data-register
empty status `$02`。data `$FFFC02`、serial bit timing、TX/RX shift registers、IKBD
firmware response、IRQ 與 MIDI ACIA 不在範圍。

- **已確認（MC6850 平台契約）**：control bits 1–0=`11` 是 master reset；status bit 1
  是 Transmit Data Register Empty（TDRE）。control 與 status 共用 register address，
  write/read 方向決定語意。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：`acia,ikbd_acia` trace 在
  FrameCycles 129,960 寫 CR=`$03` 並記錄 master reset；129,988 寫 CR=`$96`，建立
  divider=64、serial timer=1024 CPU clocks；首次 status read 在 130,192 回 `$02`。
- **已確認（固定 EmuTOS ROM）**：SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；正常 Talos
  路徑在 68,528 instructions、968,510 clocks 首次寫 `$FFFC00`，來源序列由 ROM 與
  Hatari trace共同確認為 `$03,$96`。

## typed 行為與驗收

1. cold reset／MC68000 RESET 清 control、status與 configured state。
2. 唯一允許的 control序列是 `$00→$03` master reset，再 `$03→$96` configuration；
   `$03` 將 status設為 `$02`，`$96` 保留 `$02` 並標記 configured。
3. configured 後 byte read `$FFFC00` 回 `$02`。未 configured read、錯序／重複／其他
   control、data access、user access與 word access都失敗即關閉且原子不變。
4. fixed byte access先採 4 wait clocks；更精確 ACIA E-clock／bus timing若由 oracle
   顯示不同，後續 timed-data 規格必須回填，不得默認 cycle parity。
5. synthetic test、固定 ROM、完整 240,000 筆 corpus、全測試、vet 與 build通過，
   並抵達 data-register 或其他下一 gate後才升 **CONFORMED**。

本切片不宣稱鍵盤、滑鼠或 MIDI 可用，也不改 Dungeon Master 規則、畫面或存檔。

## 驗收收據

- synthetic control/status 測試通過，非法順序、data、user access 與 word access 均失敗即關閉。
- 固定 ROM 抵達 `$FFFC02` data-register gate：68,551 instructions、4 interrupts、968,772 clocks。
- 完整 corpus、全測試、`go vet` 與建置結果記錄於專案驗證矩陣。

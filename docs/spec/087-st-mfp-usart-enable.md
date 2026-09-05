# 087 — ST MFP USART fixed serial enable

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 在 Timer D 已啟動後，將 UCR／RSR／TSR 依序寫成
`$88/$01/$01` 的 serial-port 初始狀態。實際收發、shift registers、USART interrupt、
GPIO handshake、baud output 與資料暫存器不在範圍，遇到時失敗即關閉。

- **已確認（MC68901 一手規格）**：NXP user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`。
  UCR 是字元格式／時鐘控制；RSR bit 0 是 receiver enable，TSR bit 0 是 transmitter
  enable。啟用不等同已有可收／可送資料，狀態與 interrupt source 仍是獨立狀態。
- **已確認（固定 EmuTOS 1.3 原始碼）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/serport.c:223 rsconf_mfp`
  在 baud table 驅動 `setup_timer(...,3,...)` 後，依參數順序直接寫 UCR、RSR、TSR、SCR。
  `init_serport()` 呼叫 `rsconf1(DEFAULT_BAUDRATE,0,0x88,1,1,0)`。
- **已確認（固定 ROM）**：SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；Timer D 啟動後，
  第一個未支援位址是 `$FFFA29`，寫入來源 D6=`$88`。

## typed 行為與驗收

1. TCDCR=`$51`、TDDR/main=`$02` 時，UCR `$00→$88`；接著 RSR `$00→$01`；
   最後已由軟體初始化為已知 `$00` 的 TSR `$00→$01`。每一步只提交自身 register。
2. 前置順序錯誤、其他 nonzero value、重複 write、UDR access 或任何未建模狀態都回
   `unsupported_device_state` 且原子不變。
3. cold reset／MC68000 RESET 回到規格 071 的狀態；byte read、4 wait clocks、alias、
   權限與寬度契約不變。
4. synthetic test與固定 ROM已跨過三筆 write。正常路徑抵達 68,451 instructions、
   4 interrupts、967,594 clocks；下一個 gate 是 IERA `$FFFA07` 寫 `$10`，完整 state
   為 SSP=`$0F66`、SR=`$2300`、pipeline PC=`$FC61D4`、prefetch=`$FA07,$1238`。
   完整 240,000 筆 CPU corpus、全測試、vet 與 build通過後升 **CONFORMED**。

本切片不聲稱 serial I/O 可用，也不改 Dungeon Master 規則、畫面、存檔或權利邊界。

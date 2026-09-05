# 071 — ST MFP USART reset writes

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS reset 迴圈對 MC68901 Synchronous Character Register
（SCR，`$FFFA27`）、USART Control Register（UCR，`$FFFA29`）、Receiver Status
Register（RSR，`$FFFA2B`）與 Transmitter Status Register（TSR，`$FFFA2D`）依序寫
`$00`，以及該軟體初始化後的零值讀回。非零格式、receiver/transmitter enable、
serial clock、狀態旗標、收發資料與 UDR 均不在範圍。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §3.3、§7.1.2、§7.1.3、§7.2.2、§7.3.2。硬體 reset 將 SCR、UCR、RSR
  清零並停用 receiver/transmitter；TSR 與 UDR 明列為不由硬體 reset 清除。
  RSR bit 0 是 receiver enable，TSR bit 0 是 transmitter enable，故 `$00` 寫入維持
  收發器停用。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/mfp.c:25-36
  reset_mfp_regs`，檔案 SHA-256
  `40452d84c3f743895590e412f7cacd1eba5dd5edb9d39ff2c39252471295ba30`。
  迴圈由 GPIP 每隔一 byte 寫 `$00`，上界是 `&mfp->tsr`，因此包含 SCR、UCR、
  RSR、TSR，並刻意排除寫入後可能在 Timer D 啟動時送出的 UDR。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c:137-141`
  映射五個 byte register；檔案 SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`。
  `src/rs232.c:517-610` 的四個 handler 均增加 4 wait clocks；檔案 SHA-256
  `c03dff57107cdc6989fa3dbd38b2c2e32004d2d5b06c3622cf9511d456e1faa3`。
  TSR bit 7 是唯讀的 transmit buffer empty 狀態；本專案尚未送出 UDR 時，依此硬體
  契約回傳空緩衝區狀態，而不是把 bit 7 存入可寫 control latch。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，SCR FrameCycles
  `44958→44974`、UCR `45002→45018`、RSR `45046→45062`、TSR
  `45090→45106`，各 **16 clocks**；每次寫入前後四個 register 均顯示 `$00`。

## typed 行為

1. SCR、UCR、RSR 在 cold reset 與 MC68000 `RESET` 後為已知 `$00`；TSR 因硬體
   reset 值未定而為未知，直到軟體明確寫 `$00`。
2. supervisor data byte 對四個位址只接受「目前為 `$00`／未知 TSR → 寫 `$00`」；
   成功後 register 為已知 `$00`。任何非零寫入均回
   `unsupported_device_state`，且不改狀態。
3. 只有已知值可 byte read；TSR 在軟體初始化前 read 失敗即關閉，軟體寫零後讀回
   `$80`（control=`$00`、buffer empty=`$80`）。user access 與 word access 仍 fault。
4. 四位址 byte access 各增加 4 wait clocks；EmuTOS 同形 MOVE 各 16 clocks。
5. UDR `$FFFA2F` 維持未映射；本切片不建立 serial clock、資料 buffer、IRQ 或外部 I/O。

## 驗收與停止線

- table test 涵蓋 reset known/unknown、四次零寫、讀回、alias、4 wait clocks、非零
  原子失敗、user protection、word access與 UDR 未映射。
- 固定 EmuTOS 應完成 TSR reset write，state、prefetch、時鐘與四 register 對上
  Hatari；之後以有界 probe 找出下一個 fail-closed 停止點，不預設它是 UDR。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均須通過，才可
  將狀態改為 **CONFORMED**。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari／EmuTOS 的 GPL 實作。

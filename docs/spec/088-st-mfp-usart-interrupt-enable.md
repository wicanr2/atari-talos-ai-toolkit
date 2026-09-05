# 088 — ST MFP USART interrupt channel enable

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 在 USART 已設為 UCR／RSR／TSR=`$88/$01/$01`
後，依序啟用 Receive Buffer Full（RBF）與 Transmit Buffer Empty（TBE）的 IERA／IMRA
bits。實際 USART source、pending、priority、IRQ 與 IACK 不在範圍。

- **已確認（MC68901 一手規格）**：NXP user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`；
  IERA／IMRA 的 bit 4 與 bit 2 分別 gate interrupt channels 12 與 10，enable 與 mask
  是獨立級，無 source event 時不應憑空設定 IPRA。
- **已確認（固定 EmuTOS 1.3 原始碼）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`include/biosdefs.h:189/191`
  定義 MFP_TBE=10、MFP_RBF=12；`bios/serport.c:1041/1043 init_serport()` 依序呼叫
  `mfpint(MFP_RBF,...)`、`mfpint(MFP_TBE,...)`。
- **強證據（固定 Hatari 2.4.1 trace）**：先寫 IERA／IMRA=`$10/$10`；第二次
  `mfpint()` 的清理階段保留同值 `$10`，最後寫成 `$14/$14`。此段沒有 MFP
  USART exception。

## typed 行為與驗收

1. USART fixed init完成且 IPRA=0 時，IERA 允許 `$00→$10`、`$10→$10`、
   `$10→$14`；每次只更新 enable latch，不設定 pending。
2. IMRA 沿用規格 067 的無 pending完整 latch，正常路徑最終應為 `$14`。
3. 前置狀態錯誤、其他 nonzero transition、pending 非零、重複 `$14` 或 disable
   都失敗即關閉且原子不變。reset、read、4 wait clocks、alias、權限與寬度不變。
4. synthetic test與固定 ROM已完成 IERA／IMRA=`$14/$14`。正常路徑抵達
   68,518 instructions、4 interrupts、968,318 clocks；下一 gate 是 YM2149 register
   select `$FF8800` 寫 `$07`，完整 state 為 SSP=`$0F8C`、SR=`$2300`、pipeline
   PC=`$FC6DEC`、prefetch=`$8800,$11FC`。完整 240,000 筆 corpus、全測試、vet 與
   build通過後，本規格升 **CONFORMED**。

本切片不聲稱 USART interrupt 可觸發，也不改遊戲規則、畫面、存檔或權利邊界。

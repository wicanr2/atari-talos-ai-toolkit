# 067 — ST MFP IMRA／IMRB mask latch

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Interrupt Mask Register A／B（IMRA `$FFFA13`、IMRB
`$FFFA15`）的 reset、byte read 與無 pending 狀態下的 byte write，並驗證固定
EmuTOS 初始化的兩次 `$00` write。interrupt source、IRQ output、priority、IACK
與「已有 pending 時改 mask」的 IRQ 重新評估仍未接線。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §4.3.3。IMRA／IMRB reset=`$00`；bit 0 表示 mask、bit 1 表示 unmask。
  mask 不清除 enabled channel 的 pending bit，但會立即停止該 channel 的 IRQ；
  重新 unmask 時，既有 pending 依 priority 重新請求服務。兩 register 可隨時讀取。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA13/$FFFA15` 映至 IMR byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset IMR=`$00`、write 完整保存 byte、access 增加 4 wait clocks，之後重新評估
  IRQ。此 GPL 程式只作外部 oracle，不翻譯、移植或連結。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第十、十一次迭代分別對 IMRA／IMRB 寫 `$00`；ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，IMRA iteration
  FrameCycles `44518→44534`、IMRB `44562→44578`，各 **16 clocks**；兩 register
  前後皆 `$00`，寫後 SR=`$2714`、prefetch=`$5488,$B0FC`。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 IMRA、IMRB 設為 `$00`。
2. supervisor data byte read 回各自 mask latch；user access 依 I/O protection fault，
   word access仍不合法。
3. 當對應 IPRA／IPRB 為 `$00` 時，software byte write 完整保存 value；因此可測
   `$A5→$3C→$FF→$00`，read 必須逐次相同。
4. 若對應 pending latch 非零，mask write 回 `unsupported_device_state`，且 IMR／IPR
   都不得改變。這是暫時的失敗即關閉邊界，不冒充 IRQ 重新評估已完成。
5. 兩位址 byte access各增加 4 wait clocks；EmuTOS 同形 MOVE各 16 clocks。
6. 本規格驗收時以 `$FFFA17` Vector Register write 的 reserved-I/O fault 作為
   停止線；後續規格 068 只取代這條停止線，不改變 IMR 契約。

## 驗收與停止線

- table test涵蓋 reset、完整 byte latch、alias、4 wait clocks、user protection、
  word access、pending 非零時的原子失敗及 Vector Register 未映射。
- 固定 EmuTOS 應完成第 7,515 條、累計 177,078 clocks，state、prefetch、
  IMRA／IMRB 對上 Hatari；再三條控制指令後，第 7,519 次嘗試在 `$FFFA17`
  明確停止，成功完成數維持 7,518。該停止線其後由規格 068 取代，其他未規格化
  register 仍未泛化。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari GPL 實作。

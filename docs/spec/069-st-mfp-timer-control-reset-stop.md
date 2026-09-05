# 069 — ST MFP timer control reset-stop

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Timer A Control Register（TACR，`$FFFA19`）、Timer B Control
Register（TBCR，`$FFFA1B`）與 Timer C/D Control Register（TCDCR，`$FFFA1D`）的
reset、byte read 及 reset-state `$00→$00` stop write，並驗證固定 EmuTOS 初始化。
prescaler、main counter、delay／event-count／pulse-width mode、TAO／TBO output、timer
deadline 與 interrupt 仍未接線。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §6.2.2。三個 control register reset=`$00`；control 0 表示 timer stopped，
  counting inhibited、main counter 不受影響、prescaler residual count 丟失。
  TACR／TBCR 的非零 control 可選 delay、event-count、pulse-width，bit 4 可強制
  TAO／TBO low；TCDCR 的 C/D 各三位可選 delay prescaler，bits 7、3 unused。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  映射三個 byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  對 access 增加 4 wait clocks，改 control 前更新 timer state，停止 active timer 時
  保存 current counter，並依新 control 啟停 timer。這些非零／active 路徑不在本切片。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第十三至十五次迭代依序對 TACR／TBCR／TCDCR 寫 `$00`；ROM
  SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，TACR FrameCycles
  `44650→44666`、TBCR `44694→44710`、TCDCR `44738→44754`，各 **16 clocks**；
  三 register 前後皆 `$00`，寫後 SR=`$2714`、prefetch=`$5488,$B0FC`。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 TACR、TBCR、TCDCR 設為 `$00`。
2. supervisor data byte read 回 `$00`；user access 依 I/O protection fault，word
   access仍不合法。
3. 各 register 只接受 `$00→$00` byte write。任一非零 value 或非零既有 control
   回 `unsupported_device_state`，且不得改變 state；這是完整 timer state machine
   未接線前的失敗即關閉邊界。
4. 三位址 byte access各增加 4 wait clocks；EmuTOS 同形 MOVE各 16 clocks。
5. 完成 TCDCR 後，下一次迴圈在 `$FFFA1F` Timer A Data Register write 維持
   reserved-I/O fault。

## 驗收與停止線

- table test涵蓋 reset/read、`$00→$00`、非零 write 原子失敗、alias、4 wait clocks、
  user protection、word access及 TADR 未映射。
- 固定 EmuTOS 應完成第 7,531 條、累計 177,254 clocks，state、prefetch 與三個
  control register 對上 Hatari；再三條控制指令後，第 7,535 次嘗試在 `$FFFA1F`
  明確停止，成功完成數維持 7,534。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari GPL 實作。

# 065 — ST MFP IPRA／IPRB write-zero-to-clear

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Interrupt Pending Register A／B（IPRA `$FFFA0B`、IPRB
`$FFFA0D`）的 reset、byte read 與 software clear semantics，並驗證固定 EmuTOS
初始化的兩次 `$00` write。實際 interrupt source、acknowledge、priority、IRQ output
與 in-service state仍未接線。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §4.3.2。IPRA／IPRB reset=`$00`；enabled channel 收到 interrupt 時對應 pending
  bit 置 1；vectored acknowledge 或 polled handler 可清除 pending bit。各 bit 的
  1/0 定義為 pending/cleared。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA0B/$FFFA0D` 映至 IPR byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset IPR=`$00`、access 增加 4 wait clocks，software write 採
  `pending &= written`，亦即 0 清除、1 保留，不能由 software 設 pending；之後
  重新評估 IRQ。此來源只作外部 oracle。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第六、七次迭代分別對 IPRA／IPRB 寫 `$00`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 同前。IPRA iteration
  FrameCycles `44342→44358`、IPRB `44386→44402`，各 **16 clocks**；兩 register
  前後皆 `$00`，其餘 state 變化與前述迴圈相同。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 IPRA、IPRB 設為 `$00`。
2. supervisor data byte read 回各自 pending latch；user access依 I/O protection
   fault，word access仍不合法。
3. software byte write採 `pending = pending & value`。write bit 0 清除既有 pending；
   write bit 1 保留既有值，不能把 0 設成 1。此切片可測試注入 latch，但尚無公開
   interrupt source 可將其設為 1，且不宣稱 IRQ output 已完成。
4. 兩位址 byte access各增加 4 wait clocks；EmuTOS 同形 MOVE各 16 clocks。
5. 本規格驗收時以 `$FFFA0F` ISRA write 的 reserved-I/O fault 作為停止線；
   後續規格 066 只取代這條停止線，不改變 IPR 契約。

## 驗收與停止線

- table test涵蓋 reset、read、`$A5 & $3C = $24`、write `$FF` 不設零 bit、alias、
  wait、user protection、word access及 ISRA 未映射。
- 固定 EmuTOS 完成第 7,499 條、累計 **176,902 clocks**；state、prefetch、
  IPRA／IPRB 對上 Hatari。再完成三條控制指令後，第 7,503 次嘗試在
  `$FFFA0F` ISRA 明確停止，成功完成數維持 7,502。該停止線其後由規格 066
  取代，其他未規格化 register 仍未泛化。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

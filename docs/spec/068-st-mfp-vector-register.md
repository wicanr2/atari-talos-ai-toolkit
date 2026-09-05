# 068 — ST MFP Vector Register

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Vector Register（VR，`$FFFA17`）的 reset、byte read、vector
base、automatic／software end-of-interrupt（EOI）模式切換與切回 automatic 時清除
ISRA／ISRB，並驗證固定 EmuTOS 初始化的 `$00` write。interrupt source、IRQ output、
priority、IACK 與實際 vector delivery 仍未接線。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §4.1.3、§4.4.1–§4.4.3。VR reset=`$00`；bits 7–4 是使用者提供的 vector
  高四位；bit 3（S）為 1 選 software EOI、為 0 選 automatic EOI 並強制
  ISRA／ISRB 為 0；bits 2–0 unused 且 read 為 0。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA17` 映至 VR byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset VR=`$00`、access 增加 4 wait clocks，從 software 切回 automatic 時清
  ISRA／ISRB 並重新評估 IRQ。Hatari 保存低三個 unused bits，與一手規格的
  read-zero 契約不同；本專案採一手規格。GPL 程式只作外部 oracle。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第十二次迭代對 VR 寫 `$00`；ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，VR iteration
  FrameCycles `44606→44622`，共 **16 clocks**；VR 與 ISRA／ISRB 前後皆 `$00`，
  寫後 SR=`$2714`、prefetch=`$5488,$B0FC`。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 VR 設為 `$00`，ISRA／ISRB 亦為 `$00`。
2. supervisor data byte read 回 `VR & $F8`；user access 依 I/O protection fault，
   word access仍不合法。
3. byte write 保存 `value & $F8`。bits 7–4 可任意設定 vector base；bit 3=1 選
   software EOI 且不自行設定 ISR；bit 3=0 選 automatic EOI並清除 ISRA／ISRB。
4. 若寫入 automatic EOI 時 IPRA 或 IPRB 非零，回 `unsupported_device_state`；
   VR、ISR、IPR 都不得改變。這是 IRQ 重新評估尚未接線前的失敗即關閉邊界。
5. byte access 增加 4 wait clocks；EmuTOS 同形 MOVE為 16 clocks。
6. 完成 VR 後，下一次迴圈在 `$FFFA19` Timer A Control Register write 維持
   reserved-I/O fault。

## 驗收與停止線

- table test涵蓋 reset、vector base、unused bits read zero、兩種 EOI 模式、切回
  automatic 清雙 ISR、pending 非零時的原子失敗、alias、4 wait clocks、user
  protection、word access及 TACR 未映射。
- 固定 EmuTOS 應完成第 7,519 條、累計 177,122 clocks，state、prefetch、VR、
  ISRA／ISRB 對上 Hatari；再三條控制指令後，第 7,523 次嘗試在 `$FFFA19`
  明確停止，成功完成數維持 7,522。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari GPL 實作。

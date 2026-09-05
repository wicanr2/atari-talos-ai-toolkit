# 070 — ST MFP timer data stopped-load

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Timer A/B/C/D Data Registers（TADR `$FFFA1F`、TBDR
`$FFFA21`、TCDR `$FFFA23`、TDDR `$FFFA25`）的 reset，以及對應 timer 停止時的
byte read/write 與 main-counter 同步載入；並驗證固定 EmuTOS 初始化的四次 `$00`
write。active timer、prescaler、countdown、reload、timeout、output 與 interrupt 不在範圍。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §6.2.1。四個 TDR 與 8-bit main counter reset=`$00`；read 捕捉 main counter；
  timer 停止時 write 同時載入 TDR 與 main counter。timer active 時 write 只先進
  TDR，待 counter count through `$01` 才 reload；臨界 write 可產生不定 main-counter
  值，故不在本切片猜補。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  映射四個 byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  對 access 增加 4 wait clocks，write 保存 TDR，control=stopped 時同步 main counter。
  Timer D 另有選配 BIOS patch 與 RS232 baud side effect；本切片固定 Hatari 執行結果
  為 `$00`，Talos 不移植該 emulator policy。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第十六至十九次迭代依序對 TADR／TBDR／TCDR／TDDR 寫 `$00`；
  ROM SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，TADR FrameCycles
  `44782→44798`、TBDR `44826→44842`、TCDR `44870→44886`、TDDR
  `44914→44930`，各 **16 clocks**；四 data register 前後皆 `$00`。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將四個 TDR 與四個 main counter 設為 `$00`。
2. supervisor data byte read 在 timer stopped 時回 main counter；user access 依 I/O
   protection fault，word access仍不合法。
3. 對應 timer stopped 時，byte write 原子地將任意 value 同時存入 TDR 與 main
   counter。`$00` 代表硬體的 256-count reload 編碼，但 register byte 仍讀 `$00`。
4. 對應 control 非零時，read/write 回 `unsupported_device_state`，且 TDR／main
   counter 不變；這是 active counter 捕捉／延後 reload 未接線前的停止線。
5. 四位址 byte access各增加 4 wait clocks；EmuTOS 同形 MOVE各 16 clocks。
6. 完成 TDDR 後的驗收停止點是 `$FFFA27` Synchronous Character Register write；
   該後續位址現由規格 071 接管，本規格的 timer-data 契約不變。

## 驗收與停止線

- table test涵蓋 reset、任意 byte 同步載入 TDR/main counter、`$00` round-trip、
  active control 原子失敗、alias、4 wait clocks、user protection、word access及
  `$FFFA27` 在本規格驗收當時未映射；後續接線見規格 071。
- 固定 EmuTOS 應完成第 7,547 條、累計 177,430 clocks，state、prefetch 與四個
  TDR/main counter 對上 Hatari；再三條控制指令後，第 7,551 次嘗試在 `$FFFA27`
  明確停止，成功完成數維持 7,550。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari GPL 實作。

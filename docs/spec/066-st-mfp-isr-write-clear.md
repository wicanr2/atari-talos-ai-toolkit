# 066 — ST MFP ISRA／ISRB write-zero-to-clear

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68901 Interrupt In-Service Register A／B（ISRA `$FFFA0F`、ISRB
`$FFFA11`）的 reset、byte read 與 software clear，並驗證固定 EmuTOS 初始化的
兩次 `$00` write。interrupt acknowledge（IACK）、priority、IRQ output、Vector
Register 的 EOI mode 切換及 interrupt source 仍未接線。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §4.3.4、§4.4.1–§4.4.3。ISRA／ISRB reset=`$00`；automatic EOI mode 強制
  in-service bits 為 0；software EOI mode 在 IACK 傳出 vector 時設對應 bit，
  processor 寫 0 可清除，寫 1 不改變，register 可隨時讀取。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA0F/$FFFA11` 映至 ISR byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset ISR=`$00`、access 增加 4 wait clocks，software write 採
  `in_service &= written`，之後重新評估 IRQ。此 GPL 程式只作外部 oracle，
  不翻譯、移植或連結。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第八、九次迭代分別對 ISRA／ISRB 寫 `$00`；ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 下，ISRA iteration
  FrameCycles `44430→44446`、ISRB `44474→44490`，各 **16 clocks**；兩 register
  前後皆 `$00`。兩次寫入前 D=`$1E,$02,0,0,$80000,$100000,5,1`，A1=`$3156`、
  A5=`$FC01F4`、A6=`$0FFC`、SSP=`$0F8C`，寫後 SR=`$2714`、prefetch=
  `$5488,$B0FC`。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 ISRA、ISRB 設為 `$00`。
2. supervisor data byte read 回各自 in-service latch；user access 依 I/O protection
   fault，word access仍不合法。
3. software byte write採 `in_service = in_service & value`：bit 0 清除既有
   in-service，bit 1 保留既有值，不能由 software 將 0 設成 1。測試可注入 latch
   以驗證此局部轉換，但 production 尚無 IACK 路徑能設位。
4. 兩位址 byte access各增加 4 wait clocks；EmuTOS 同形 MOVE各 16 clocks。
5. 本規格驗收時以 `$FFFA13` IMRA write 的 reserved-I/O fault 作為停止線；
   後續規格 067 只取代這條停止線，不改變 ISR 契約。

## 驗收與停止線

- table test涵蓋 reset、read、`$A5 & $3C = $24`、write `$FF` 不設零 bit、alias、
  wait、user protection、word access及 IMRA 未映射。
- 固定 EmuTOS 應完成第 7,507 條、累計 176,990 clocks，state、prefetch、
  ISRA／ISRB 對上 Hatari；再三條控制指令後，第 7,511 次嘗試在 `$FFFA13`
  明確停止，成功完成數維持 7,510。該停止線其後由規格 067 取代，其他未規格化
  register 仍未泛化。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均已通過。

## 玩家路徑、存檔與權利邊界

此切片只延伸固定 ROM 的可重現開機路徑，不改 Dungeon Master 規則、畫面或存檔。
NXP 手冊與固定公開原始碼只保存雜湊、定位與導出的 typed 契約；專案不收錄 TOS ROM，
也不複製 Hatari GPL 實作。

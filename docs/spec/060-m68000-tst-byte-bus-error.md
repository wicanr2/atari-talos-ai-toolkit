# 060 — MC68000 `TST.B (An)` 來源讀取 bus error

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理 `TST.B (An)` 的 byte source read 收到 backend typed bus fault 時進
vector 2，直接解開 EmuTOS 1.3 對未啟用 Blitter 的 `$FF8A3C` 探測。

- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMem.c` SHA-256
  `ba67024fc35deeed202276e95844effbc014327f0893af71b85102601d2103cd`。
  `MACHINE_ST` 且 Blitter disabled 時，初始化將 `$FF8A00–$FF8A3F` 設為 bus-error
  region。`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  識別 `$FF8A3C` 為 Blitter control byte register；本切片不實作該裝置。
- **已確認（固定 EmuTOS 1.3 原始碼）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  `detect_blitter()` 用 `check_read_byte(BLITTER_CONFIG1)` 探測 Blitter，bus error
  表示不存在。
- **已確認（Hatari 外部 oracle）**：image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS 1.3 UK ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`、
  `--machine st --memsize 1 --blitter false --fast-boot false`。故障前 `$FC0636`
  opcode=`$4A10`、A0=`$FFFF8A3C`、SSP=`$0F84`、SR=`$2704`、
  prefetch=`$4A10,$4E71`、FrameCycles=35524；vector 2=`$FC063C`。
  handler 入口 FrameCycles=35588，差值 **64 clocks**；SSP=`$0F76`，prefetch
  =`$21C9,$0008`。

## typed 行為

1. `TST.B (An)` 的 backend error 只有在實作 `BusFault`、read、size=1 且 fault address
   等於 24-bit effective address 時才轉成 vector 2；其他錯誤繼續失敗即關閉。
2. 故障 read transaction 使用偶數 bus address `$FF8A3C`、size=1、FC=5、UDS=true、
   LDS=false；奇數 byte lane 行為尚未由本切片驗證，不泛化。
3. 14-byte frame 自 SSP `$0F76` 起為
   `$4A15,$FFFF,$8A3C,$4A10,$2704,$00FC,$0638`：SSW、32-bit fault address、
   opcode、原 SR、saved PC。saved PC 是下一指令 `$FC0638`。
4. vector fetch 使用 FC=5，handler prefetch 使用 FC=6；D/A、USP 與 condition codes
   保持故障前值，CPU 進 supervisor 並清 trace。
5. 此 `(An)` source fault 固定 64 clocks。其他 byte EA、byte write、word／long、
   instruction fetch、double fault 與 Blitter enabled 行為不在本切片。

## 驗收與停止線

- synthetic ST memory 整合測試以 `$FF8A3C` reserved I/O 觸發 fault，檢查 clocks、
  fault transaction lane、完整 frame、state、vector fetch 與 handler prefetch。
- 固定 EmuTOS 完成第 6,917 條探測，對上 Hatari 的 frame／state／64 clocks；
  再有界步進找下一停點。
- 完整 230,000 筆 CPU 語料、固定 ROM、Go 測試、靜態檢查與建置均需通過，才升
  **CONFORMED**。

## CONFORMED 收據

- synthetic ST memory 測試確認 64 clocks、byte fault transaction 的 UDS lane、
  vector 2 fetch、handler prefetch、完整 frame 與 register preservation。
- 固定 EmuTOS 第 6,917 條完成後，Atari Talos 為 168242 累計 clocks；D/A、
  SSP=`$0F76`、SR=`$2704`、PC=`$FC0640`、prefetch=`$21C9,$0008` 與 Hatari
  handler 入口一致，frame 七個 words 全同。
- 後續成功完成 7,474 條；下一停點是對 `$FFFA01` 的 MFP byte write，移交周邊
  裝置規格，不以 void register 或 RAM 冒充。

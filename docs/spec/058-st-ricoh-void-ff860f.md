# 058 — Atari ST Ricoh `$FF860F` void byte read

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理首版 `MACHINE_ST`／Ricoh chipset 對 `$FF860F` 的 supervisor byte read，
解開 EmuTOS 1.3 `detect_modectl()` 的機型探測。它不是 DMA mode/control register 實作。

- **已確認（Atari ST 公開硬體契約）**：Atari ST DMA 位址 byte registers 是
  `$FF8609/$FF860B/$FF860D`；普通 ST 沒有 `$FF860F` Falcon mode/control register。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMem.c` SHA-256
  `ba67024fc35deeed202276e95844effbc014327f0893af71b85102601d2103cd`，
  `IoMem_FixVoidAccessForST()` 將 `$FF860F` 設成不產生 bus error 的 void access，
  `IoMem_VoidRead()` 將這類 read 回傳 `$FF`。`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`；一般 ST
  register table 只列到 `$FF860D`。
- **已確認（固定 EmuTOS 1.3 原始碼）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`。`bios/dma.h` SHA-256
  `cc52f9a325ad44caeb0657249d3f73d8f41aacd6cec6f98fd11a626b3de1da0f`
  將 `$FF860F` 定義成僅 Falcon 存在的 `modectl`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`
  的 `detect_modectl()` 用 `check_read_byte()` 探測它。
- **已確認（Hatari 外部 oracle）**：image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS 1.3 UK ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`、
  `--machine st --memsize 1 --fast-boot false`。`$FC0636` 的 `TST.B (A0)` 前
  A0=`$FFFF860F`、SR=`$2704`、prefetch=`$4A10,$4E71`、FrameCycles=34720；
  執行後到 `$FC0638` 為 34728，SR=`$2708`、prefetch=`$4E71,$7001`，沒有例外。

## typed 行為

1. `Memory.ReadByte($FF860F, supervisor data FC=5)` 回 `$FF`，不產生 bus fault。
2. 24-bit address masking 保留，因此 `$FFFF860F` 與 `$00FF860F` 指向同一 access。
3. `TST.B (A0)` 沿用既有 MC68000 行為：結果 `$FF` 令 N=1、Z=0，X 保留，8 clocks。
4. user-mode I/O protection仍先於位址特例，user data FC=1 必須回 `FaultProtected`。
5. `$FF860F` write、word/long access、Mega-ST IMP chipset、STE／MegaSTE／Falcon 的
   register 行為與其他 void I/O 位址均不在本切片，維持既有失敗即關閉。

## 驗收與停止線

- ST memory 測試涵蓋 24-bit alias、supervisor `$FF` read、user protection、相鄰
  `$FF860E/$FF8610` 仍為 reserved I/O fault，以及 write 仍失敗即關閉。
- 固定 EmuTOS 從 reset 成功完成第 6,851 條 `TST.B (A0)`；檢查 8 clocks、D/A、
  SSP、SR、PC 與 prefetch，然後以有界步進找下一個第一停點。
- 通過全部 CPU 語料、固定 ROM、Go 測試、靜態檢查與建置後升 **CONFORMED**。

## CONFORMED 收據

- memory 測試通過 `$00FF860F`／`$FFFF860F` read `$FF`、user protection、相鄰
  reserved I/O fault 與 write 失敗即關閉。
- 固定 EmuTOS 第 6,851 條為 8 clocks；完成後 D/A、SSP=`$0F84`、SR=`$2708`、
  PC=`$FC063C`、prefetch=`$4E71,$7001` 對上 Hatari tracepoint。
- 後續成功執行到第 6,878 條；下一停點 `$FFFC21` 已由規格 059 接手。

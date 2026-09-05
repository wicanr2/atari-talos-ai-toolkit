# 057 — MC68000 line-F emulator／vector 11

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68000 執行 `$F000–$FFFF` line-F opcode 時進入 vector 11，直接解開
EmuTOS 1.3 在 `$FC00BE` 的 CPU／FPU 探測。

- **已確認（NXP 官方 MC68000 契約）**：既有固定《M68000 Family Programmer's
  Reference Manual》將 vector 11／offset `$02C` 定義為 F-Line Emulator；PDF
  SHA-256 `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（MAME microcoded 外部語料）**：固定 `SingleStepTests/m68000` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5` 的 `ILLEGAL_LINEF.json.bin` 共
  2,500 筆，保存 initial／final CPU、RAM、clocks 與完整 bus transaction。
- **已確認（Hatari 外部 oracle）**：固定 Hatari 2.4.1 image 與 EmuTOS ROM；
  line-F 前為 18 條／496 clocks、PC=`$FC00C2`、prefetch=`$F010,$0800`，
  vector 11=`$00FC00D4`，含 ST 外部匯流排相位等待後進 handler 為 36 clocks。

## typed 行為

1. decoder 將所有且僅 `$Fxxx` opcode 路由 vector 11；其他未實作 opcode仍失敗即關閉。
2. saved PC 是 opcode address `State.PC-4`；沿用已驗證的 6-byte format-0 frame、
   supervisor stack bank、trace clear、FC=5 vector reads 與 FC=6 handler prefetch。
3. MC68000 核心 exception 基準為 34 clocks；ST 在目前 496 clocks 的 phase 0
   起點會因例外內部交易相位補 2 clocks，因此整機為 36 clocks。成功後 machine
   instruction count 加 1。
4. line-A／vector 10、原生 `$4AFC` ILLEGAL、FPU 實作與 exception double fault 不在本切片。

## 驗收

- `ILLEGAL_LINEF.json.bin` 2,500 筆 state／RAM／clocks／bus transaction 全同。
- 固定 EmuTOS 第 19 條由 496→532，到 `$FC00D4` handler；frame、SSP、SR、PC 與
  prefetch 對上 Hatari，再找下一個第一停點。
- 全 CPU 語料累計 230,000 筆；Go 測試、靜態檢查與建置保持通過。

## 驗收結果

- 固定 MAME microcoded `ILLEGAL_LINEF.json.bin` 2,500 筆全部通過；核心基準
  34 clocks，state、RAM 與七筆 bus transaction 全同。
- 固定 EmuTOS 第 19 條由 496→532 clocks，進 `$FC00D4`，SSP=`$0FE0`；frame
  保存 SR=`$2700`、PC=`$00FC00BE`，prefetch=`$21FC,$00FC`。
- 向後有界探測成功執行 6,850 條；下一停點是 `$FF860F` 保留 I/O，已移交下一切片。

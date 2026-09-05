# 051 — MC68000 對 68010+ MOVEC 的 illegal-instruction vector 4

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68000 執行 68010+ `MOVEC Rc,Rn`／`MOVEC Rn,Rc`
opcodes `$4E7A`、`$4E7B` 時的 illegal-instruction vector 4，直接解開
EmuTOS 1.3 在 `$FC0070` 的 CPU 型號探測。

- **已確認（NXP 官方 MC68000 契約）**：《M68000 Family Programmer's
  Reference Manual》Appendix B 將 vector 4／offset `$010` 定義為 Illegal
  Instruction；68010 才有 VBR／`MOVEC`，ST／STF 的 MC68000 不執行它。
  PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false`。
- `$FC0070` 執行前為 7 條指令／92 clocks，`SSP=$1000`、`SR=$2704`、
  prefetch=`$4E7B,$0801`；vector 4 long 為 `$00FC0074`。
- handler `$FC0074` 邊界為 8 條指令／128 clocks，因此本例外花
  36 clocks。`SSP=$0FFA`、`SR=$2704`，D0–D7／A0–A6 不變；
  handler prefetch=`$21FC,$00FC`。MMU cold `$00` 下，logical `$0FFA` 映射到
  physical `$1FFA`，frame words 為 `$2704,$00FC,$0070`，證明 saved PC
  是 opcode 位址 `$FC0070`，不是 next PC。

## typed 行為

1. decoder 只將 `$4E7A`、`$4E7B` 在 MC68000 模式路由到 vector 4。
   其他尚未實作但對 MC68000 合法的 opcode 仍失敗即關閉，不可全部
   偽裝成 illegal instruction。
2. saved PC 為 `State.PC-4`（opcode 位址），以原 SR 在 supervisor
   stack 建立 6-byte format-0 frame；記憶體排列為 SR、PC high、PC low。
3. bus 寫入次序為 PC low、SR、PC high；再以 supervisor data FC=5
   讀 vector `$10/$12`，以 supervisor program FC=6 讀 handler 前兩 words。
4. 成功後 SSP 減 6，S 設 1、trace bit 清 0，其餘 SR 保留；PC 依
   Atari Talos next-prefetch 契約為 handler+4，prefetch 是 handler 前兩 words。
5. 本例外固定 36 clocks。machine 只在成功時將 instruction count 加 1、
   clock count 加 36。

## 驗收與停止線

- synthetic bus 同時驗 supervisor／user 前態、SSP／USP bank、trace clear、
  saved opcode PC、frame 內容、vector／prefetch FC、bus 次序與 36 clocks。
- 固定 EmuTOS ROM 必須從 reset 完成第 8 條指令，以 128 clocks 到達
  `$FC0074` handler，並與 Hatari 的 state／frame／prefetch 一致。
- 通過後繼續執行到新的第一失敗點；不因 vector 4 成功就宣稱
  TOS 可開機。
- 原生 `$4AFC` ILLEGAL、line-A／line-F emulator、其他 illegal encoding、
  exception 進入期再 bus fault／double fault、68010+ VBR 與 `MOVEC` 正常執行
  不在本切片。

## CONFORMED 收據

- 2026-09-05：`$4E7A`／`$4E7B` synthetic 測試通過 supervisor／user、
  trace clear、SSP／USP、saved opcode PC、format-0 frame、全 bus 次序與 36 clocks。
- 固定 EmuTOS ROM 的 `$4E7B` 後，Atari Talos 與 Hatari 均在第 8 條指令、
  128 clocks 到 handler `$FC0074`；`SSP=$0FFA`、`SR=$2704`、
  prefetch=`$21FC,$00FC`，frame=`$2704,$00FC,$0070`。
- 完整 Go 測試含 227,500 筆既有 CPU 外部語料、靜態檢查與建置全數通過。
- 後續探針雖兩邊均在 220 clocks 到 `$FC0088` `RESET`，但 Hatari
  中間 bus-error frame 保存 `$FFFF8006`，Atari Talos 當時為 `$00FF8006`。
  **該缺口已由 spec 052 關閉**（2026-09-05）：frame 現在保存 CPU 內部的 32-bit
  有效位址，端到端跑同一顆 ROM 到 `$FC0080` 得到 `$FFFF8006`。`$FC0088` 的
  `RESET` 本身仍待規格，所以那一點仍不宣稱完全對拍。

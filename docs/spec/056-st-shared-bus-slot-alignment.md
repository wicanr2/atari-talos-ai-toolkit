# 056 — Atari ST CPU 外部 bus slot 對齊

狀態：**CONFORMED**（`MOVE.L immediate→absolute-short` 正常偶數 destination 切片）。

## 問題與範圍

固定 EmuTOS 1.3 UK ROM 的第 14 條
`MOVE.L #$FC00B2,$0010` 從全機 clock 390 開始。Hatari 用 26 clocks 完成，
Atari Talos 的固定 MC68000 timing 為 24 clocks。先前暫稱此差異為 Shifter
arbitration；Hatari cycle-exact 原始碼與相鄰 phase 探針顯示，更精確的機制是
CPU external memory access 必須對齊 ST 的四 clock bus slot。它不是 destination RAM
位址特例，也不是目前已證實的即時 Shifter 搶占事件。

本切片只處理 ST／STF 8 MHz CPU external bus 的 slot 對齊。Blitter ownership、
video fetch 排程、STE／TT fast RAM、MFP E-clock 與個別 I/O register wait state
均不在本規格。

## 證據

- **已確認（外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  第 13 條後兩邊皆為 390 clocks；第 14 條後 Hatari=416、Talos=414。
- **已確認（相鄰 phase 外部探針）**：以同一 EmuTOS 基底只替換 reset 後測試區；
  phase 0 ROM SHA-256 `b834e233034c9db71c93cd0252bc4dcbe37595b18d8eabefdc829ba3b4496d89`，
  以 12-clock `JMP` 到 `$FC0040`；phase 2 ROM SHA-256
  `2cd11eb308696b4f03b6d2af272bee291fdeb921226aeb23caaf2cc71306a71f`，
  以 10-clock `BRA.W` 到同址。兩案進入 `$21FC #$FC00B2,$0010` 時 D／A／SR、PC、
  prefetch、operand 與 destination 相同；Hatari cycle-exact 分別為 12→36（24）與
  10→36（26）clocks。額外等待只由起始 phase 決定。
- **強證據（Hatari v2.4.1 固定原始碼）**：官方 GitHub mirror commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`，`src/cpu/custom.c` SHA-256
  `0cae1a5afd0548bcf63ca5f4f20d795a95f62fb882a0b270c6a0aba65d47da45`。
  `wait_cpu_cycle_read_ce020` 的 lines 217–223 與
  `wait_cpu_cycle_write_ce020` 的 lines 360–366 均先計算
  `(CyclesGlobalClockCounter + instruction-local cycles) & 3`；若 bit 1 為 1，
  先推進 `4-bus_pos` clocks，再執行實際 read／write。
- **強證據（Hatari 註解）**：`src/includes/m68000.h` SHA-256
  `41c2da3d0f1bb08c8e9fef4d9a391eff51890ee66dca09bc3b428dccd31f8fef`，
  lines 338–344 明載 cycle-exact 模式以每次 memory access 對齊四 clocks，並由此
  自然得到 instruction pairing。lines 244–265 記錄在 STF 實測的 misaligned
  access penalty，但列出的 addressing mode 不是本次 immediate→absolute-short。
- **輔助強證據**：`src/stMemory.c` SHA-256
  `2781f71b024a84bd32e6fc70102d6673da141c1e0462ac8b7113562e82a854b0`，
  lines 790–802 將 system／Shifter 共用 RAM 與沒有每 250 ns bus penalty 的 fast RAM
  明確區分。該段描述 TT topology，只能支持「共用 RAM 有 bus penalty」的硬體背景，
  不能單獨證明 ST 第 14 條的精確公式。

Hatari 為 GPL 外部 oracle；本專案只記錄可觀測契約與證據定位，不複製其程式碼。

## typed 規則

對一個準備在絕對 clock `t` 開始的 ST CPU external bus access：

```text
phase = t mod 4
wait  = 4 - phase   if phase bit 1 is set
        0           otherwise
```

MC68000 本切片只接受偶數 clock，因此輸出為 phase 0→0、2→2；若 phase 1／3 出現，
視為上游 clock-domain 違約並失敗即關閉，不從 Hatari 整數運算猜補。
wait 必須在 read／write 前插入，並推移同指令所有後續 access。

`$21FC` 有六個連續 4-clock external phases：兩次 immediate pipeline fetch、一次
absolute-short extension fetch、兩次 destination word write、一次 sequential prefetch。
固定 MAME microcoded MOVE.L 語料未抽到 `$21FC`，但相同 immediate source 加單一
destination extension 的 `$2B7C`／`$257C` 皆為
`r4,r4,r4,w4,w4,r4`、總計 24 且無 idle；Hatari phase 0 探針的 24 clocks 封閉了
額外 internal phase 的空間。因此第一個 access offset=0，其餘依序為 4、8、12、16、20；
phase 2 的 2-clock wait 插在第一個 access 前，後續 offsets 連帶成 6、10、14、18、22。

## 實作與驗收

1. 對齊屬 CPU external bus access，應由 ST timed Bus 一致處理 program ROM、RAM、
   cartridge 與 I/O 前的共同 slot；個別 I/O 額外 wait 仍由其後續裝置規格疊加。
2. 先遷移 `MOVE.L immediate→absolute-short` 的六個 phases；phase 0／2 synthetic 測試
   分別須為 24／26 clocks，transaction data／FC／address 與 final state 相同。
3. 固定 EmuTOS 必須重現第 14 條 390→416，且 RAM `$10..$13=$00FC00B2`、next prefetch
   與 Hatari 相同；不得寫 `$0010`、opcode `$21FC` 或 clock 390 的 wait 特例。
4. 本切片通過只能升該 addressing form 為 CONFORMED；其他 MOVE／CPU 指令仍逐族遷移。

## CONFORMED 收據

- 2026-09-05：ST timed Bus 對所有 CPU external byte／word access 依絕對 clock 對齊；
  phase 0／2 分別回 0／2 clocks，phase 1／3 失敗即關閉。規則不含 opcode、`$0010`
  或 clock 390 特例。
- `MOVE.L immediate→absolute-short` 以六個連續 phases 接線。synthetic phase 0／2
  測試分別得到 24／26 clocks；transaction kind／FC／address／data、RAM 結果、PC、
  prefetch 與 flags 一致，timeline 為 `0,4,8,12,16,20` 及
  `idle@0,access@2,6,10,14,18,22`。
- 固定 EmuTOS 第 14 條由 390→416；RAM `$10..$13=$00FC00B2`，state PC=`$FC00AC`、
  prefetch=`$203C,$0000`、SR=`$2700`，與 Hatari 收據一致。
- 繼續到第 18 條的逐步總 clocks 為 416／428／464／488／496；line-F 前 state
  PC=`$FC00C2`、prefetch=`$F010,$0800`、D0=`$00000808`、A0=`$FC0152`、
  SSP=`$0FE6`、SR=`$2700`。新的第一停點確定為尚未實作的 line-F／vector 11。
- 完整 227,500 筆既有 CPU 語料、Go 測試、固定 ROM 測試、靜態檢查與建置通過。
  Odd destination 的 exception timeline 與其他 addressing forms 不在本次 CONFORMED 範圍。

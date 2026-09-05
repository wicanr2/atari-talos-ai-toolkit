# 056 — Atari ST 共用記憶體 bus slot 對齊

狀態：**DRAFT**（對齊公式有強證據；第 14 條各 access offset 尚未確認）。

## 問題與範圍

固定 EmuTOS 1.3 UK ROM 的第 14 條
`MOVE.L #$FC00B2,$0010` 從全機 clock 390 開始。Hatari 用 26 clocks 完成，
Atari Talos 的固定 MC68000 timing 為 24 clocks。先前暫稱此差異為 Shifter
arbitration；Hatari cycle-exact 原始碼顯示更精確的機制是 CPU memory access
必須對齊 ST 的四 clock 共用 bus slot，Shifter 是共享該 RAM bus 的裝置背景，
不是目前已證實的即時搶占事件。

本切片只處理 ST／STF 8 MHz 共用 RAM bus 的 slot 對齊。Blitter ownership、
video fetch 排程、STE／TT fast RAM、MFP E-clock 與個別 I/O register wait state
均不在本規格。

## 證據

- **已確認（外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  第 13 條後兩邊皆為 390 clocks；第 14 條後 Hatari=416、Talos=414。
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

## 候選 typed 規則（尚未 READY）

對一個準備在絕對 clock `t` 開始的 ST shared-RAM access：

```text
phase = t mod 4
wait  = 4 - phase   if phase bit 1 is set
        0           otherwise
```

在目前只出現偶數 CPU clocks 的路徑，候選輸出為 phase 0→0、2→2；phase 1／3
仍須由完整 clock-domain 契約決定是否可達，不先從 Hatari 整數運算照抄。
wait 必須在 read／write 前插入，並推移同指令所有後續 access。

## READY 前缺口與驗收

1. 取得第 14 條每個 immediate fetch、destination write 與 prefetch 的 instruction-local
   offset；固定 MOVE.L 語料未抽到 opcode `$21FC`，不可用另一 addressing mode 代填。
2. 用至少 phase 0 與 phase 2 的同一指令外部探針驗證：同 state／operand／address，
   只改前置時相；預期差異分別為 0 與 2 clocks。
3. 確認對齊適用的地址分類：shared RAM、ROM、cartridge 與 I/O 不可因共用 Memory
   backend 而套用同一 wait。
4. 證據完成後才可升 READY，遷移 `$21FC` 的全部 access、加入 phase 表格測試，並
   以固定 EmuTOS 重現 390→416；不得寫 `$0010`、opcode `$21FC` 或 clock 390 特例。

# 013 — Motorola 68000 MOVE.B 記憶體目的端

狀態：**CONFORMED**（2026-09-05）。

## 範圍與輸入

本規格接續 CONFORMED spec 012，涵蓋 `MOVE.B <source>,<memory destination>`：

- source：spec 012 的全部合法 byte source EA；
- destination：`(An)`、`(An)+`、`-(An)`、`d16(An)`、`d8(An,Xn)`、
  absolute word 與 absolute long；
- 同一 address register 同時作 source／destination 時，目的 EA 必須看到 source
  postincrement／predecrement 已完成的值。

輸入為 spec 003 固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5` 之 `MOVE.b.json.bin`；2,500 筆中
384 筆屬 spec 012，剩餘 2,116 筆為本規格分母。

不含 MOVE.W／MOVE.L／MOVEA、bus error，或 word／long data address error。byte 存取
不觸發 alignment error；其他例外不得由本規格推定。

## 證據與等級

- **已確認（平台規格）**：Motorola／NXP《M68000 Family Programmer's Reference
  Manual》4-116～4-118；MOVE source 寫入 destination，N／Z 依 byte 結果、V／C
  清除、X 不變；destination 只允許 data-alterable modes。
- 官方 PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（獨立單步實驗）**：固定語料提供每筆 initial／final CPU state、RAM、
  clocks 與 bus transactions；TAS／TRAPV 的上游疑義不涉及本指令。

## typed 行為

1. source 讀取、source An side effect 與 FC／lane 規則沿用 spec 012。
2. destination EA 在 source 完成後計算。`(An)+` 成功寫入後更新；`-(An)` 在寫入前
   更新。A0–A6 byte delta 為 1，A7 為 2，A7 依 S bit 選 USP／SSP。
3. destination byte write 使用 user／supervisor data FC 1／5；bus address lines 為
   偶數 word base，UDS／LDS 選 high／low byte lane；只改 RAM 的目標 byte。
4. destination displacement／index extension 位於所有 source extension 之後；EA 必須
   用當下 register state 計算。目的端沒有 PC-relative 或 immediate mode。
5. flags 依實際寫出的 8-bit value 更新，其他 Dn／An 與非作用中的 stack pointer 保留。

## clocks

以同一 source 至 Dn 的 spec 012 clocks 為基準：`(An)`、`(An)+`、`-(An)` 加 4；
`d16(An)` 與 abs.w 加 8；`d8(An,Xn)` 加 10；abs.l 加 12。

## bus／prefetch 次序

- source 的 extension 與 data read 先依 spec 012 完成，但最後一格 instruction refill
  由目的端排程接手。
- `(An)`／`(An)+`：byte write → final refill。
- `-(An)`：final refill → byte write。
- d16／abs.w：consume destination extension → byte write → final refill。
- indexed：2 internal clocks → consume destination extension → byte write → final refill。
- abs.l：Dn 或 immediate source 先 consume high／low destination extensions，再 write，
  最後 refill；記憶體 source 則讀 low extension、write，再讀兩個 refill words。
  兩條路徑最終 state 可相同，但 transaction 全序不同，必須分別驗收。

## 失敗模式、影響與停止線

- 非法 destination mode、byte An-direct source 與未實作 opcode 必須失敗即關閉。
- 本工作只擴充 CPU／bus，不改 save、遊戲資產、公開 JSON Lines 契約或權利邊界。
- 只有 2,116 筆完整 state、RAM、clocks 與 bus transaction 全通過，且既有 30,384
  筆無回歸，才能升為 **CONFORMED**。不得以抽樣或只比 final state 取代。

## 驗收結果

2026-09-05：記憶體目的端 2,116／2,116 筆通過；連同 spec 012 的 Dn 目的端，
`MOVE.b.json.bin` 全部 2,500／2,500 筆通過。每筆均比較完整 CPU state、稀疏 RAM
零值正規化、clocks 與 bus transaction 全序。CPU 外部單步驗收累計 32,500 筆。

# 015 — Motorola 68000 MOVE.L 與分段 long data access

狀態：**CONFORMED**（2026-09-05）。

## 範圍與輸入

本規格涵蓋完整 `MOVE.L <source>,<destination>`：source 為全部 68000 data modes
（含 An direct），destination 為 Dn 與全部 data-alterable memory modes。輸入為 spec 003
固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5` 之 `MOVE.l.json.bin` 2,500 筆。

語料固定分母為：正常 1,013、source read address error 869、destination write address
error 618。排除 MOVEA、MOVEM、MOVEP、bus error 與 exception handler 自身再 fault。

## 證據與等級

- **已確認（平台規格）**：Motorola／NXP《M68000 Family Programmer's Reference
  Manual》4-116～4-118；long source 寫入 destination，N／Z 依 32-bit 結果、V／C
  清除、X 不變；source 允許 An direct。
- 官方 PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（固定單步實驗）**：2,500 筆語料逐筆提供 CPU state、RAM、clock 與 bus
  transaction；1,487 筆 odd data address 另提供 vector 3 的 14-byte frame。

## typed 行為

1. Dn／An direct 取完整 32 bits；Dn destination 覆寫完整暫存器。正常完成後 N 依 bit 31、
   Z 依完整結果、V／C 清除，X 與 SR 其他 bits 保留。
2. memory long access 只要求 word alignment；偶數但非四倍數的位址合法。read／write 依
   big-endian 順序在 `EA`、`EA+2` 各做一次 Size=2、UDS=LDS=true 的 bus cycle。
3. data FC 依 user／supervisor 為 1／5；PC-relative source 使用 program FC 2／6。
4. source／destination postincrement 只有完整 access 成功後才 An+4；source predecrement
   在 access 前先 An-4，destination predecrement 先以 An-4 計算 EA、完整寫入時才提交
   An-4。同 register alias 時，destination EA 看見已成功 source 的 side effect。
5. normal clocks 為 `4 + sourceCost + destinationCost`。source cost：direct 0、`(An)`／
   `(An)+` 8、`-(An)` 10、d16／abs.w／PC-d16 12、indexed／PC-indexed 14、abs.l 16、
   immediate 8。destination cost：Dn 0、`(An)`／`(An)+`／`-(An)` 8、d16／abs.w 12、
   indexed 14、abs.l 16。
6. prefetch／write 排程依語料 transaction 全序驗收：destination predecrement 在兩次
   write 前先做 final refill；memory source 配 absolute-long destination 使用 low extension、
   兩次 write、兩次 refill 的特殊排程。

## data address-error 契約

1. odd source 產生 `re`，且不做任何一半的 long read；odd destination 產生 `we`，且不做
   任何一半的 long write。fault bus address 對齊偶數，Size=2、UDS=LDS=true，data
   正規化為 0。
2. source `re` 不更新 MOVE flags。register／immediate source 的 fault-time CCR 依目的
   微操作階段而異：mode 2／3 保留原 CCR，mode 4／7 依完整 long 更新 NZVC，mode 5／6
   只依 long 更新 N／Z、保留 V／C。memory source 配 mode 2／3 時依低 word 更新 NZVC，
   或 absolute-long 特殊排程時依低 word 更新 NZVC，配 mode 4／5／6／abs.w 時已依
   完整 32-bit 結果更新 NZVC；X 一律保留。
3. source／destination postincrement 在 fault 時不生效；source predecrement 的 An-4 保留，
   destination predecrement 的 An-4 不提交；其第一個微操作 fault address 是 An-2。
4. SSW、14-byte frame、supervisor 切換、vector 3 與 handler prefetch 沿用 spec 007／014；
   SSW 使用 fault FC，read fault 另設 `0x0010`。
5. source saved PC 相對初始 PC：mode 2／3／4／5／6 與 PC-relative 為 -2，abs.w 為 0，
   abs.l 為 +2。source clocks 為 54 加 word-EA cost：4／6／8／10／12。
6. destination saved PC 已包含 source extension：direct 與 mode 2／3／4 為 0，d16／indexed／
   abs.w／PC-relative 為 +2，abs.l 與 immediate 為 +4；absolute-long destination 在
   non-memory source 另依管線加 2。
7. destination clocks 為 58 加 long source cost，再加 destination fault extra：mode 2／3
   為 0，predecrement／d16／abs.w 為 4，indexed 為 6；abs.l 對 non-memory／memory
   source 分別為 8／4。

## 影響、驗收與停止線

- 只擴充 CPU long data access，不改 JSON Lines、save、Atari 素材或第三方權利邊界。
- 逐筆比較 2,500 筆完整 state、RAM、clocks、bus transaction、`re`／`we` 與 exception
  frame；既有 35,000 筆不得回歸。
- 三個固定分母全部通過才能升為 **CONFORMED**；不得用 aligned-only 或 Dn-only 子集宣稱完成。
- 2026-09-05 驗收結果：正常 1,013／1,013、source `re` 869／869、destination `we`
  618／618，合計 2,500／2,500；全套外部單步語料累計 37,500 筆全數通過。

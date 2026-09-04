# 012 — Motorola 68000 MOVE.B 來源 EA 至 Dn

狀態：**CONFORMED**（2026-09-05）。

## 範圍

本規格只涵蓋 `MOVE.B <source>,Dn`。目的端固定為 D0–D7，來源涵蓋 MC68000
允許的全部 byte data addressing modes：Dn、`(An)`、`(An)+`、`-(An)`、
`d16(An)`、`d8(An,Xn)`、absolute word／long、`d16(PC)`、`d8(PC,Xn)` 與 immediate。

不含記憶體目的端、MOVE.W、MOVE.L、MOVEA、data address error 或 bus error；這些
仍須另立 READY 規格，不得由本規格推定已完成。

## 平台規格證據

- Motorola／NXP《M68000 Family Programmer's Reference Manual》，
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- 手冊 4-116～4-118：source 寫入 destination；N／Z 依結果、V／C 清除、X 不變；
  byte source 不允許 address-register direct，其餘上述 source modes 均允許。

## 外部單步證據

- 使用 spec 003 固定的 `SingleStepTests/m68000` commit，從 `MOVE.b.json.bin`
  2,500 筆中依 opcode destination mode `000` 篩出 384 筆。
- 每筆比較完整 CPU state、RAM、clocks 與 bus transactions；篩選分母固定為 384，
  避免測試因條件漂移而靜默少跑。
- 語料確認手冊未規定的 prefetch、function code、byte lane、EA 更新及 microcode 時序。

## 行為契約

1. 來源 byte 寫入目的 Dn 的低 8 bits，高 24 bits 保留。
2. N 依 bit 7、Z 依 8-bit 結果；V／C 清除，X 與 SR 其他 bits 保留。
3. user／supervisor data space 分別用 FC 1／5；PC-relative source 用 program FC 2／6。
4. byte bus 的 address lines 輸出偶數 word base；偶數語意位址以 UDS 選 high byte，
   奇數語意位址以 LDS 選 low byte。`SparseMemory.ReadByte` 仍以原語意位址取值。
5. `(An)+` 在成功讀取後更新，`-(An)` 在讀取前更新；A0–A6 的 byte delta 為 1，
   A7 的 byte delta 為 2。A7 依 S bit 指向 USP 或 SSP。
6. d16 與 brief-index displacement 均做符號延伸；PC-relative base 為 extension word
   位址。index word／long 與 A7 解讀沿用 CONFORMED spec 010。
7. 立即值取 extension word 的低 byte；address-register direct 必須失敗即關閉。

## clocks 與 bus 順序

| source | clocks | 可見 bus 順序 |
|---|---:|---|
| Dn | 4 | program refill |
| `(An)`、`(An)+` | 8 | data byte read → program refill |
| `-(An)` | 10 | internal 2 clocks → data byte read → program refill |
| d16(An)、abs.w、d16(PC) | 12 | extension refill → data byte read → program refill |
| d8 indexed | 14 | internal 2 clocks，extension refill → data byte read → program refill |
| abs.l | 16 | low extension → first refill → data byte read → second refill |
| immediate | 8 | first refill → second refill |

## 驗收

2026-09-05：384／384 筆全部通過；加上先前 30,000 筆，CPU 外部單步驗收累計
30,384 筆。這只代表本文件明列的 slice 已 CONFORMED，不代表完整 MOVE.B 或 MOVE family。

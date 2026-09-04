# 014 — Motorola 68000 MOVE.W 與 data address error

狀態：**CONFORMED**（2026-09-05）。

## 範圍與輸入

本規格涵蓋完整 `MOVE.W <source>,<destination>`：source 為全部 68000 data modes
（含 An direct），destination 為 Dn 與全部 data-alterable memory modes。輸入為 spec 003
固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5` 之 `MOVE.w.json.bin` 2,500 筆。

語料分母固定為：正常 1,013、source read address error 839、destination write address
error 648。三者合計必須等於 2,500。排除 MOVEA、MOVE.L、bus error 與 exception
handler 自身再 fault；不得由本規格推定其行為。

## 證據與等級

- **已確認（平台規格）**：Motorola／NXP《M68000 Family Programmer's Reference
  Manual》4-116～4-118；word source 寫入 destination，N／Z 依 16-bit 結果、V／C
  清除、X 不變；word source 允許 An direct。
- 官方 PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（固定單步實驗）**：語料逐筆提供 CPU state、RAM、clock 與 bus transaction；
  1,487 筆 odd data address 另提供 vector 3 的 14-byte frame 與 handler prefetch。

## 正常 typed 行為

1. Dn／An direct 取低 16 bits；Dn destination 只覆寫低 16 bits。memory read／write
   使用 user／supervisor data FC 1／5；PC-relative source 使用 program FC 2／6。
2. 所有 memory word address 必須為偶數；word bus 以 UDS＋LDS 傳完整 16 bits。
3. source postincrement 在 read fault 前已令 An+2；destination postincrement 只有 write
   成功才令 An+2。predecrement 先令 An-2 再 access。A7 依 S bit 選 USP／SSP；同
   register alias 時，destination EA 看見 source side effect。
4. extension、index、PC-relative base、prefetch 與 normal bus 排程沿用已 CONFORMED 的
   MOVE.B 規格，僅 operand 寬度與 An-direct source 不同。
5. 正常完成後 N 依 bit 15、Z 依 16-bit 結果，V／C 清除，X 與 SR 其他 bits 保留。

## data address-error 契約

1. odd source 產生 `re`；odd destination 產生 `we`。fault bus address 對齊偶數，
   Size=2、UDS=LDS=true，data 欄視為未定義並在 corpus loader 正規化為 0。
2. `re` 不更新 MOVE flags；`we` 在 exception frame 前依 source word 更新 flags。
3. SSW 為 `(opcode & 0xffe0) | FC`，read fault 另加 `0x0010`。frame 仍依既有
   vector-3 14-byte 格式保留 opcode、fault address、saved PC 與 fault 時 SR。
4. source saved PC 相對初始 PC：`(An)`／`(An)+`／d16／indexed／PC-relative 為 -2；
   `-(An)`／abs.w 為 0；abs.l 為 +2。
5. destination saved PC 為 source extension 已消耗後的 PC；absolute-long destination
   若 source 非 memory，再加 2。destination final refill 不納入 saved PC。
6. source `re` clocks：58，加 predecrement 2、one-extension 4、indexed 6、abs.l 8。
   destination `we` clocks：58 加 source normal extra cost，再加 destination
   predecrement／d16／abs.w 4、indexed 6；abs.l 在 non-memory source 加 8、memory
   source 加 4。
7. fault 後用 supervisor FC 5 取 vector 3；SR 設 S、清 T，其他 bits 取 fault 時 SR；
   handler 用 program FC 6 預取兩 words。exception transaction 全序必須完全一致。

## 影響、驗收與停止線

- 只擴充 CPU／word bus，不改 JSON Lines、save、Atari 素材或第三方權利邊界。
- 需增加 word write 與例外單元測試，並逐筆比較 2,500 筆完整 state、RAM、clocks、
  `re`／`we`、frame 及 bus transaction；既有 32,500 筆不得回歸。
- 2026-09-05 驗收結果：正常 1,013／1,013、source `re` 839／839、destination `we`
  648／648，合計 2,500／2,500；全套外部單步語料累計 35,000 筆全數通過。

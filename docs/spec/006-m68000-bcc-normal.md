# 006 — Motorola 68000 Bcc／BRA 正常控制流

狀態：**CONFORMED**（2026-09-05）。

## 平台規格證據

- Motorola／NXP《M68000 Family Programmer's Reference Manual》，
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，`M68000PRM.pdf`，
  SHA-256 `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- Bcc：手冊 4-25～4-26；BRA：4-55。位移基準都是分支指令字位址加 2，條件碼不變。
- 68000 的 8-bit 位移為 opcode 低 byte；低 byte 為 0 時改用緊接的 16-bit extension。
  本階段不實作 68020 才有的 32-bit 位移。

## 外部單步證據與範圍

- 使用 spec 003 固定的 `SingleStepTests/m68000` commit 與 `Bcc.json.bin`。
- 2,500 筆中，1,830 筆為正常偶數目標：1,147 筆 byte 條件不成立、586 筆 byte
  條件成立、83 筆 byte BRA、9 筆 word 條件不成立、5 筆 word 條件成立。
- 其餘 670 筆含 `re` address-error transaction，必須由例外堆疊規格處理；本規格不准
  把它們當正常分支，也不准靜默對齊位址。
- ReDMCSB DM12EN 重建組語大量使用 BRA 與 Bcc，因此這是 Dungeon Master 控制流主幹。

## 行為

1. 條件碼 2～15 依 C、V、Z、N 的 68000 真值表判斷；BRA（condition 0）永遠成立。
2. byte 位移以 signed 8-bit 延伸；word 位移以預取的 extension 作 signed 16-bit 延伸。
3. 成立時，以「初始 next-prefetch PC 減 2」為規格所稱的指令字位址加 2，再加位移；
   清空舊預取並從目標及目標加 2 各讀一個 word。
4. byte 條件不成立時丟棄 opcode、保留舊 `prefetch[1]`，再從初始 PC 補一個 word。
5. word 條件不成立時亦丟棄 extension，從初始 PC 與初始 PC 加 2 重建兩格預取。
6. SR、資料／位址暫存器、USP、SSP 與 RAM 不變。
7. 正常成立分支為 10 clocks；byte 不成立為 8 clocks；word 不成立為 12 clocks。

## 失敗與排除

- condition 1 是 BSR，必須走堆疊寫入規格，不屬於本規格。
- 奇數目標必須明確回傳尚未實作的 address error；在完整例外規格 READY 前，不得改
  CPU 狀態冒充原版例外。
- bus read 失敗原樣回傳，不得留下半套成功結果。

## 驗收

從固定 Bcc 語料辨識並逐筆驗收 1,830 筆不含 `re`／`we` 的正常案例，比較完整 CPU
狀態、RAM、總 clock 與 bus transaction。通過前不得升為 **CONFORMED**。

2026-09-05 驗收結果：1,830 筆正常案例全部通過。奇數分支目標後續已由 CONFORMED
的 spec 007 實作 address error，不再停留於暫時拒絕狀態。

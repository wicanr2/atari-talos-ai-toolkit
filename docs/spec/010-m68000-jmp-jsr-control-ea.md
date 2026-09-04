# 010 — Motorola 68000 JMP／JSR control effective address

狀態：**CONFORMED**（2026-09-05）。

## 平台規格證據

- Motorola／NXP《M68000 Family Programmer's Reference Manual》，
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- control addressing modes 與 PC-relative 規則見 2.2；68000 brief extension 見 2-21；
  JMP 見 4-108，JSR 見 4-108～4-109。
- 68000 支援 `(An)`、`d16(An)`、`d8(An,Xn)`、`(xxx).W`、`(xxx).L`、`d16(PC)`、
  `d8(PC,Xn)`。brief index 可為 Dn／An 與 word／long；scale bits 在 68000 上忽略。

## 外部單步證據與範圍

- 使用 spec 003 固定的 `SingleStepTests/m68000` commit；`JMP.json.bin` 與
  `JSR.json.bin` 各 2,500 筆，完整涵蓋上述七種 control modes、user／supervisor、
  正常偶數目標與奇數目標 address error。
- 語料確認 extension 消耗、absolute-long 額外讀取、PC-relative base、index
  sign-extension、bits 10–8 忽略、clock、預取、JSR stack write 次序及
  address-error saved PC。

## 共用 effective-address 行為

1. `(An)` 直接取 An；A7 依執行前 S bit 對應 USP／SSP。
2. `d16(An)` 為 An 加 signed extension word。
3. `d8(An,Xn)` 為 An 加 signed extension low byte，再加 index；word index 先 sign-extend，
   long index 取完整 32 bits。68000 不實作後代 CPU 的 scale／full-extension 解碼，
   並依官方相容性說明忽略 extension bits 10–8，不因其非零產生例外。
4. `(xxx).W` 將 extension word sign-extend；`(xxx).L` 以已預取 high word 加上從目前 PC
   讀取的 low word組成 32-bit address。
5. `d16(PC)`／`d8(PC,Xn)` 的 PC base 是 extension word 位址，即初始 next-prefetch
   PC 減 2。
6. 一個 extension word 的 return PC 為初始 PC；absolute long 為初始 PC 加 2；
   `(An)` 為初始 PC 減 2。

## JMP 行為

1. 偶數目標清空舊預取並從 target／target+2 讀取；依 mode 為 8、10、12 或 14 clocks。
2. 奇數目標進入 spec 007，saved PC 為初始 PC 減 2；依 mode 為 58、60、62 或
   64 clocks。
3. 不修改 active stack 或 condition codes。

## JSR 行為

1. 奇數目標在任何 stack write 前進入 spec 007；saved PC 是該 mode 的 return PC。
2. 偶數目標先讀 target 第一個 word，接著以原 data FC 將 return PC high／low 推入
   active stack，最後讀 target+2；依 mode 為 16、18、20 或 22 clocks。
3. condition codes 不變；只有 active stack pointer 減 4。

## 驗收

兩份語料共 5,000 筆必須逐筆比較完整 CPU 狀態、RAM、總 clock 與 bus transaction；
全部通過後才能升為 **CONFORMED**。

2026-09-05 驗收結果：JMP 的 1,272 筆正常與 1,228 筆 address error、JSR 的
1,341 筆正常與 1,159 筆 address error 全部通過。另以單元測試確認非 control mode
失敗即關閉。

## 停止線

- 不接受非 control mode；extension bits 10–8 則依 68000 行為忽略。
- Bus fault、double fault 與 68020 full extension 另立規格。

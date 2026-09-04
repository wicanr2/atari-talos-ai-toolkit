# 007 — Motorola 68000 指令讀取 address error

狀態：**CONFORMED**（2026-09-05）。

## 平台規格證據

- Motorola／NXP《M68000 Family Programmer's Reference Manual》，
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- 附錄 B 表 B-1 指定 address error 為 vector 3、offset `0x00000C`；圖 B-2 指定
  MC68000 的 14-byte bus／address-error stack frame。

## 外部單步證據與範圍

- 使用 spec 003 固定的 `SingleStepTests/m68000` commit；`Bcc.json.bin` 有 670 筆
  taken branch 對奇數指令位址取 word 的 `re` 案例。
- 全 670 筆共同確認 fault transaction、frame 欄位、實際寫入順序、user→supervisor
  切換、trace bit 清除、vector fetch、handler 預取及 60 clocks。
- 本規格只涵蓋 Bcc／BRA 造成的指令讀取 address error；資料讀寫、其他指令預取錯誤、
  bus error、例外處理期間再次故障與 double-bus-fault halt 另立規格。

## 行為

1. taken branch 的目標為奇數時，以原模式 program function code 記錄一次 `re`；
   address bus 使用目標清除 bit 0 的值。此 cycle 不 assert AS、不得呼叫 Bus；data bus
   欄位未定義，Atari Talos 與驗收載入器一律正規化為 0，不比較語料中的隨機殘值。
2. 保存的 PC 是分支 opcode 後的位址，即初始 next-prefetch PC 減 2；保存的 SR 是
   例外前原值，instruction register 是分支 opcode，access address 是未對齊目標。
3. 切入 supervisor mode、清除 trace bit，使用 SSP 並將它減 14；USP 不變。
4. 14-byte frame 自新 SSP 起依序為：special status word、access address long、
   instruction register、舊 SR、saved PC long。
5. 已定義的 SSW 低 bits 為 read bit、instruction bit 與原 function code。語料的高
   11 bits 帶有 opcode residue；為完整語料相容而保存，但不宣稱軟體可依賴未定義 bits。
6. frame 的實際 bus write 次序依外部語料固定為 saved-PC low、舊 SR、saved-PC high、
   instruction register、access-address low、SSW、access-address high，全部使用 FC 5。
7. 以 FC 5 從 `0x00000C`／`0x00000E` 讀 vector 3 handler，再以 FC 6 從 handler 與
   handler 加 2 建立預取；最終 PC 為 handler 加 4，總計 60 clocks。

## 驗收

完整跑 `Bcc.json.bin` 2,500 筆；spec 006 的 1,830 筆正常控制流與本規格的 670 筆
address error 都必須逐筆比較完整 CPU 狀態、RAM、總 clock 與 bus transaction。

2026-09-05 驗收結果：670 筆 address error 與 1,830 筆正常控制流在同一次完整語料
執行中全部通過。`re` 的未定義 data bus 欄位依本規格正規化後不參與虛假的精確比較。

## 停止線

- 例外 frame 寫入或 vector fetch 再次失敗時，MC68000 的 halt 行為不在本規格內；Bus
  錯誤必須明確回傳，不得假裝完成例外。
- 不由 MAME 語料宣稱未定義 SSW 高 bits 是所有實體 68000 的穩定契約。

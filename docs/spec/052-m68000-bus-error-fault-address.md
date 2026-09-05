# 052 — MC68000 匯流排錯誤 frame 的 32-bit access address

狀態：**CONFORMED**（2026-09-05）。

## 範圍與停止線

本切片只處理一件事：bus-error frame 裡的 **access address 欄位保存哪個值**。

MC68000 內部的位址是 32-bit，外部只接出 A1–A23（A0 由 `UDS`／`LDS` 取代），所以
**匯流排上看得到的位址天生就是 24-bit 的**。frame 裡的 access address 不來自匯流排，
來自 CPU 內部算出的有效位址，因此不受那 24 條線限制。absolute short 定址把 16-bit
延伸值符號擴展成 32-bit，`$8006` 因此是 `$FFFF8006`；同一個週期送上匯流排的是
`$FF8006`。**兩個值都對，差別在誰在看。**

不在本切片：`MOVE.W` 以外的指令與寬度、寫入方向的 bus error、instruction fetch
的 bus error、例外處理期間再次故障與 double bus fault。那些各自逐片驗收，本切片
只改 access address 的取值來源，不擴大 bus error 的涵蓋面。

## 平台規格證據

- Motorola／NXP《M68000 Family Programmer's Reference Manual》，
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
  附錄 B 圖 B-2 的 MC68000 14-byte bus／address-error stack frame 把 access address
  定義成 **long**，與 saved PC 同寬。
- **本專案內部已 CONFORMED 的同欄位**：spec 007（address error）第 4 條把 frame 排列
  定為「special status word、access address long、instruction register、舊 SR、
  saved PC long」，而它保存的是 CPU 算出的未對齊目標位址，沒有先過 24-bit 遮罩，
  670 筆外部語料逐筆通過。bus error 與 address error 共用同一個 frame 版面與同一支
  `enterAccessError`，**同一個欄位不能有兩種取值規則**。

## Hatari oracle

- spec 051 的 CONFORMED 收據已記下這個差異：EmuTOS 1.3 UK 192 KiB ROM
  （`ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`）
  在 `$FC0080` 執行 `TST.W $8006`，兩邊都在 220 clocks 到達 `$FC0088`，但
  **Hatari 的中間 bus-error frame 保存 `$FFFF8006`，Atari Talos 保存 `$00FF8006`**。
- 該收據同時說明了 `$FC0088` 因此不宣稱完全對拍。本切片的目的就是關掉這個缺口。

## typed 行為

1. bus-error frame 的 access address 是 **CPU 內部算出的 32-bit 有效位址**，
   不是送上匯流排的值，也不是記憶體實作回報的值。
2. 24-bit 遮罩只作用於兩件事：實際呼叫 `Bus` 的位址，以及記入 transaction 的位址。
   兩者都代表匯流排上真正發生的事，維持現狀。
3. 記憶體實作透過 `BusFault` 回報的位址只用來**核對**，不用來填 frame：它必須等於
   CPU 有效位址遮罩 24-bit 之後的值。不相等表示 CPU 與記憶體對「剛才存取的是哪裡」
   有分歧，此時回傳錯誤而不是挑一個值用——這種分歧若被吞掉，frame 會是自洽但錯的。
4. function code、read／write 與存取寬度仍取自 `BusFault`，本切片不改。

## 驗收與停止線

- synthetic：absolute short 的負值延伸（`$8006` → `$FFFF8006`）在 frame 的
  access address long 必須逐位元相符，且同一次存取記入 transaction 的位址是
  `$FF8006`——**同一個測試同時檢查兩個值**，否則「改成 32-bit」會退化成「不再遮罩」。
- synthetic 負對照：位址暫存器帶高位元（如 `A0 = $01FF8006`）的 `(A0)` 存取，
  frame 保存 `$01FF8006`，匯流排仍是 `$FF8006`。
- 記憶體回報位址與 CPU 有效位址不一致時必須失敗，不得靜默採用任一邊。
- 既有 227,500 筆 68000 外部語料、UCSD 直譯器驗收與靜態檢查全數不得回歸。
- 本切片通過**不宣稱** `$FC0088` 之後的 EmuTOS 開機已對拍；那要等 `RESET` 規格。

## CONFORMED 收據

- 2026-09-05，synthetic：`TST.W $8006` 的 frame access address 為 `$FFFF8006`，
  同一步的 fault transaction 位址為 `$FF8006`，PC 為 handler＋4。
- 2026-09-05，負對照：`TST.W (A0)` 且 `A0 = $01FF8006` 時 frame 保存 `$01FF8006`、
  匯流排仍是 `$FF8006`——高位元來自暫存器而非符號擴展，所以「乾脆不遮罩」這種改法
  在前一條會過、在這條會被抓出來。
- 2026-09-05，端到端：真實 EmuTOS 1.3 UK ROM
  （`ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`）
  從 reset 走到 `$FC0080` 的 `TST.W $8006`，frame 保存 `$FFFF8006`，
  與 spec 051 記錄的 Hatari 值相同。
- 三條測試都做過鑑別力驗證：把取值改回記憶體回報的位址，三條全數失敗且失敗值是
  `$00FF8006`——正是 spec 051 收據記下的 Atari Talos 舊值。
- 227,500 筆 68000 外部語料、UCSD 直譯器驗收與全套 Go 測試在同一次執行中通過。
  語料那側不受影響：`SparseMemory` 不實作 `BusFault`，`errors.As` 因此不成立，
  bus-error 路徑走不到。

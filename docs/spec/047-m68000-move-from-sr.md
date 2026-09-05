# MC68000 `MOVE SR,<ea>` 規格

狀態：**CONFORMED**

## 範圍與證據

- 指令：`MOVE SR,Dn` 與 `MOVE SR,<data alterable memory>`。
- 固定語料：`MOVEfromSR.json.bin`，2,500 筆。
- 語料 SHA-256：`c6a416d1e93c31d1901a1296f08e7cb14e99cf8dbe5c6541878b1956751fb725`。
- 上游版本：`SingleStepTests/ProcessorTests` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5`。
- 分布：Dn 404 筆、奇數記憶體目的位址錯誤 978 筆、正常記憶體目的 1,118 筆。

## 可實作契約

1. **已確認**：來源是完整 16-bit SR；寫入 Dn 時只取代低 word、保留高 word，且
   指令不修改 SR。
2. **已確認**：Dn 目的型只做順序預取，固定 6 clocks。
3. **已確認**：記憶體目的型只接受 mode 2–6 與 absolute word／long。它先讀目的
   word，再完成順序預取，最後寫入 SR；SR 不因舊目的值改變。總 clocks 是 8 加
   既有 word memory EA cost，並保留 postincrement／predecrement 與 A7 bank 行為。
4. **已確認**：奇數 word 目的位址進入 vector 3，沿用既有讀取型位址錯誤 frame；
   不發生成功寫入，但 EA 已發生的副作用依語料保留。
5. **已確認**：MC68000 的本指令在 user mode 合法，不產生 privilege violation。

## 排除範圍

- backend 回報的 bus fault、double fault 與尚未完成的 ST I/O 裝置不在本切片。
- 68010 之後 `MOVE CCR,<ea>` 與不同 privilege 契約不在 MC68000 首版。

## 驗收

- 2,500 筆固定語料的最終 CPU state、RAM、clocks 與 bus transactions 必須逐筆全同。
- 全專案測試、`go vet -stdmethods=false` 與 CLI build 必須通過。

結果：2,500／2,500 筆全同；全專案測試、vet 與 build 亦通過。CPU 外部單步
驗收累計 225,000 筆。

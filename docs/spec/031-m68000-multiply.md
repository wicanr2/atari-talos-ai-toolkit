# 031 — Motorola 68000 MULS／MULU

狀態：**CONFORMED**（2026-09-05）。

## 範圍與輸入

涵蓋 MC68000 的 `MULS.W <ea>,Dn` 與 `MULU.W <ea>,Dn`。驗收輸入為 spec 003
固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5`：

- `MULS.json.bin`：2,500 筆。
- `MULU.json.bin`：2,500 筆。

68020 的 long multiply、除法與乘加不在本切片；不合法 EA 失敗即關閉。

## 證據與 typed 行為

- **已確認（平台規格）**：Motorola／NXP《M68000 Family Programmer's Reference
  Manual》；官方 PDF SHA-256 為
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- MULS 將來源 EA 低 16 位與目的 Dn 低 16 位視為 signed，產生 32-bit 結果；MULU
  將兩者視為 unsigned。結果覆寫完整 Dn。
- 32-bit 結果設定 N／Z，V／C 清除，X 保留。
- 來源支援 MC68000 合法 word data EA；word odd address 進入 vector 3，正常 EA 的
  function code、extension、side effect 與 prefetch 必須逐筆符合固定語料。
- 執行 clocks 包含 EA 成本與原始 MC68000 資料相依乘法迭代成本；不能以固定延遲
  取代。MULU 為基底 38 加來源 word 每個 1-bit 的 2 clocks；MULS 為基底 38，
  從虛擬前一位 0 開始，來源 bit 0 至 bit 15 每次 0↔1 轉換增加 2 clocks。
- **已確認（DM 使用範圍）**：DM12EN 產生組語中 MULS 324 次、MULU 77 次，合計
  401 個靜態使用點；31 份 `.S` 雜湊清單 SHA-256 為
  `607feb2bc104af411d13e9909d9c57ffff626c3ef8737ec5cd765b22a7dafd46`。

## 失敗模式與驗收

An direct、非法 mode、long multiply、除法、未實作 opcode 與 bus backend error 必須
回傳錯誤。不改 JSONL 控制協定、Atari 素材、存檔或權利邊界。兩份語料共 5,000 筆
逐筆比較完整 state、RAM、clock、bus 與 exception frame；既有 130,000 筆不得回歸。
2026-09-05 驗收結果為 5,000／5,000 全數通過；全套外部單步語料累計 135,000 筆
全數通過。

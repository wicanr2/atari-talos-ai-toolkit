# 024 — Motorola 68000 LINK／UNLK

狀態：**CONFORMED**（2026-09-05）。

## 範圍與輸入

本規格涵蓋 `LINK An,#<displacement>` 與完整 `UNLK An`，包含正常與 odd-frame
vector-3 路徑。輸入為：

- spec 003 固定的 `SingleStepTests/m68000` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5` 中 `LINK.json.bin` 2,500 筆。
- Motorola／NXP《M68000 Family Programmer's Reference Manual》的 LINK／UNLK 定義。
- 同一固定來源中的 `UNLINK.json.bin` 2,500 筆；先前因檔名不是助記碼 `UNLK` 而漏列。

排除 bus error、例外處理器自身再次 fault 及 68020 的 `LINK.L`。UNLK odd frame 的
vector-3 exception 微時序改由 `UNLINK.json.bin` 審查，不再以 fail-closed 作完成狀態。

## 證據與等級

- **已確認（平台規格）**：LINK 先把指定 An long push 到 active stack，以 push 後 SP
  建立 frame pointer，再把 16-bit displacement 符號延伸後加到 SP；UNLK 先令 SP=An，
  從該處 pop long 回 An，最後令 SP 前進 4。兩者不改 CCR。
- 官方 PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（固定單步實驗）**：LINK 2,500 筆逐筆比較完整 CPU state、RAM、clock 與
  bus transaction，涵蓋 A0–A7、user／supervisor active stack、正負 displacement、
  push 與 prefetch 次序。
- **已確認（固定單步實驗）**：`UNLINK.json.bin` 含正常 1,385 筆、odd-frame vector-3
  1,115 筆。正常路徑固定檢查兩次 data read、An／SP 結果、active stack、12 clocks、
  program prefetch 與 CCR 不變。
- **已確認（固定單步實驗）**：odd frame 在 active SP 或 An 尚未提交前，以原始
  supervisor 狀態選 data FC 1／5 產生 read address error；fault address 為完整 frame，
  saved PC 為目前順序 PC，總計 58 clocks。
- **已確認（Dungeon Master 使用範圍）**：ReDMCSB DM12EN 產生組語中 LINK、UNLK
  各 445 次，合計 890 個靜態使用點。這只用於實作優先序，不等同執行期頻率。31 份
  `.S` 逐檔 SHA-256 清單的 SHA-256 為
  `607feb2bc104af411d13e9909d9c57ffff626c3ef8737ec5cd765b22a7dafd46`。

## typed 行為

1. LINK 消耗 signed word displacement；依目前 S bit 選 USP／SSP，先 big-endian push
   指定 An，再把 push 後 SP 寫入 An，最後以 displacement 調整 active SP。
2. UNLK 先把指定 An 當作 active SP，依 big-endian 讀出 long，寫回指定 An，再把
   active SP 設為 frame+4。An=A7 是已確認的 alias 特例：讀出的 long 是最後提交值，
   因而覆蓋中間的 frame+4，最終 active SP 等於讀出的 long。
3. LINK 為 16 clocks；UNLK 正常路徑為 12 clocks。兩者不改 Dn 或 CCR，並在資料
   存取後補滿 program prefetch。
4. UNLK frame 為奇數時，不提交 `SP=An`，直接以該 frame、原始 data FC 與目前 PC
   建立 vector-3 frame；切換 supervisor 後只由 exception 更新 SSP／SR／PC／prefetch。

## 失敗模式、影響與驗收

- nil bus、backend error 與未實作 opcode 必須回傳錯誤。
- 不改 JSONL 控制協定、Atari 素材、存檔或權利邊界。
- LINK 2,500／2,500 維持通過；UNLINK 正常 1,385、odd-frame vector-3 1,115，
  合計 2,500 筆完整 state、RAM、clock、bus 與 exception frame 全同。外部單步語料
  累計 157,500 筆，故本規格升為 CONFORMED。

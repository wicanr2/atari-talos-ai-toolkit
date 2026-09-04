# 024 — Motorola 68000 LINK／UNLK

狀態：**CONFORMED**（2026-09-05；UNLK odd-frame vector-3 微時序不在本規格範圍）。

## 範圍與輸入

本規格涵蓋 `LINK An,#<displacement>` 與偶數 frame address 的 `UNLK An`。輸入為：

- spec 003 固定的 `SingleStepTests/m68000` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5` 中 `LINK.json.bin` 2,500 筆。
- Motorola／NXP《M68000 Family Programmer's Reference Manual》的 LINK／UNLK 定義。
- UNLK 的固定本地 state／RAM／clock／bus 正常路徑測試；上游固定語料沒有 UNLK 檔。

排除 UNLK odd frame address 的 vector-3 exception 微時序、bus error、例外處理器自身再次
fault 及 68020 的 `LINK.L`。odd frame 必須失敗即關閉，不得冒充成功。

## 證據與等級

- **已確認（平台規格）**：LINK 先把指定 An long push 到 active stack，以 push 後 SP
  建立 frame pointer，再把 16-bit displacement 符號延伸後加到 SP；UNLK 先令 SP=An，
  從該處 pop long 回 An，最後令 SP 前進 4。兩者不改 CCR。
- 官方 PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（固定單步實驗）**：LINK 2,500 筆逐筆比較完整 CPU state、RAM、clock 與
  bus transaction，涵蓋 A0–A7、user／supervisor active stack、正負 displacement、
  push 與 prefetch 次序。
- **已確認（本地契約測試）**：UNLK 偶數 frame path 固定檢查兩次 data read、An／SP
  結果、active stack、12 clocks、program prefetch 與 CCR 不變。因缺獨立 UNLK corpus，
  此項不冒稱外部單步 oracle。
- **已確認（Dungeon Master 使用範圍）**：ReDMCSB DM12EN 產生組語中 LINK、UNLK
  各 445 次，合計 890 個靜態使用點。這只用於實作優先序，不等同執行期頻率。31 份
  `.S` 逐檔 SHA-256 清單的 SHA-256 為
  `607feb2bc104af411d13e9909d9c57ffff626c3ef8737ec5cd765b22a7dafd46`。

## typed 行為

1. LINK 消耗 signed word displacement；依目前 S bit 選 USP／SSP，先 big-endian push
   指定 An，再把 push 後 SP 寫入 An，最後以 displacement 調整 active SP。
2. UNLK 先把指定 An 當作 active SP，依 big-endian 讀出 long，寫回指定 An，再把
   active SP 設為 frame+4。An=A7 時最後 SP 仍是 frame+4。
3. LINK 為 16 clocks；UNLK 正常路徑為 12 clocks。兩者不改 Dn 或 CCR，並在資料
   存取後補滿 program prefetch。

## 失敗模式、影響與驗收

- nil bus、backend error、UNLK odd frame、未實作 opcode 必須回傳錯誤。
- 不改 JSONL 控制協定、Atari 素材、存檔或權利邊界。
- LINK 2,500／2,500 外部語料全數通過；UNLK 正常與 odd fail-closed 本地測試通過；
  既有 82,500 筆不得回歸。外部單步語料累計 85,000 筆。

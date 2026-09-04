# 023 — Hatari 外部 oracle 收據

狀態：**DRAFT**（2026-09-05；公開契約載體待使用者定案）。

## 已確認的需求與證據

- Hatari 只能是程序外 oracle，不得成為 Go library dependency，也不得複製、翻譯、
  連結或移植其 GPL 程式碼。
- 每次對拍必須固定並記錄 Hatari、容器 image、機型、RAM、TOS、磁碟、輸入序列與
  檢查點；原版 TOS、磁碟、快照及截圖預設不進 Git。
- 主機現有 `sundog-atari-st-oracle:20260812`：image digest
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  Hatari 2.4.1，含 Xvfb、xdotool、Python 3。它可作第一個隔離執行環境，但尚未證明
  Dungeon Master 的正常入口。
- SunDog 經實跑修正過的證據 JSON 顯示，至少需要來源雜湊、工具版本、輸入邊界、
  事件、檢查點 artifact 雜湊、證據等級、未知項、下一閘門、保留政策與勘誤鏈。

## 待定根決策

公開且穩定的收據契約載體尚未定案：

1. **目錄 bundle**：一份 canonical manifest 加相對路徑 artifact；適合截圖、memory dump、
   trace 與 diff，且可在 Git 只提交 manifest／雜湊。
2. **單一 JSONL 事件流**：容易直接串給 LLM、Codex、Claude Code 與 shell，但大型 artifact
   只能外部引用，重建完整一次 run 較仰賴事件排序規則。
3. **bundle 為 canonical、JSONL 為等價串流投影**：同時支援保存與互動控制，但需維護
   manifest／event projection 的雙向一致性測試。

在使用者確認前，本規格不定義正式 schema，不新增 production parser／writer。

## 不受載體選擇影響的邊界

- 所有 artifact 都需 `sha256`、byte length 與 media type；影像另記 width、height、pixel
  format 與擷取方式。
- 原版與 Talos 檢查點必須有相同的 checkpoint ID，並分開保存 observed state；compare
  結果不可覆蓋原始觀測。
- 未取得的欄位不得以零值冒充；未知與未支援必須可區分。
- 收據必須能標示 `confirmed`、`strong_inference`、`hypothesis`、`unknown`，並保留
  supersession／correction 關係。
- 輸入只能由正常 Hatari／Atari Talos 控制介面送入；debugger 可擷取狀態，不可用 guest
  RAM 注入假造正常玩家路徑。

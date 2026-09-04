# Atari Talos 目前狀態

更新日期：2026-09-05。

## 已定案

- 專案名為 Atari Talos AI Toolkit；repo 為 `atari-talos-ai-toolkit`，CLI 為 `ataritalos`。
- 目標是讓 LLM、Codex、Claude Code、`go test` 與 shell 能決定性控制 Atari ST 原版，
  產生 remake 同狀態對拍證據。
- 第一階段為無頭 Atari ST／STF，不含 STE、TT、Falcon。
- Go library 與 CLI 共用協定型別；CLI 的穩定公開介面是 JSON Lines。
- Hatari 只作外部 oracle，不使用其 GPL 程式碼。
- 授權採 RRSAL-1.0；原版 TOS、磁碟與遊戲素材由使用者自備。

## 現況

- M0 控制契約已建立：`hello`、`capabilities`、`quit` 可用。
- `run_frames` 等需要機器核心的命令明確回傳 `not_implemented`。
- public repo 已建立：<https://github.com/wicanr2/atari-talos-ai-toolkit>，預設分支 `main`。
- 尚未實作 68000 或任何 Atari ST 硬體，不宣稱可開機或執行遊戲。

## 下一步

1. 固定第一批 Motorola 68000 公開測試語料及其授權、版本與雜湊。
2. 撰寫 68000 暫存器、例外、匯流排與週期契約的 DRAFT 規格。
3. 建立最小指令解碼垂直切片；通過獨立語料後才擴充 opcode。
4. 另建 Hatari 外部 oracle 收據格式，不讓 Hatari 成為 library dependency。

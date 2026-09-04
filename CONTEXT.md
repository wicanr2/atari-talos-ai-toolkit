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
- Motorola 68000 第一條垂直切片已完成：NOP 的狀態、預取、4 clocks 與 program bus
  read 通過 SingleStepTests/MAME 產生的 2,500 筆外部語料；其餘指令未實作。
- 第二個指令族 MOVEQ 也通過 2,500 筆；覆蓋 D0–D7、立即數符號延伸、X 保留及
  N／Z／V／C 更新。CPU 外部語料目前合計 5,000 筆。
- 尚未實作完整 68000 或 Atari ST 周邊硬體，不宣稱可開機或執行遊戲。

## 下一步

1. 依 NOP 已驗證的 pipeline／bus 模型，逐組擴充 68000 opcode；每組先寫 READY 規格。
2. 撰寫例外、address error、interrupt acknowledge 與 reset vector 的 DRAFT 規格。
3. 另建 Hatari 外部 oracle 收據格式，不讓 Hatari 成為 library dependency。
4. 建立 ST／STF memory map，先走 EmuTOS reset／開機的最小路徑。

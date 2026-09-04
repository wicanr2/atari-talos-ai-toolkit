# Atari Talos 工作歷程

## 2026-09-05

- 建立專案骨架、RRSAL-1.0、繁中／英文入口與 Docker Go 工具鏈。
- 建立 `talos-jsonl/1` 控制契約及 CLI；未完成模擬能力採失敗即關閉。
- 固定使用既有 `golang:1.24-bookworm`，明示 `/usr/local/go/bin` 後全套測試與 CLI
  實際往返通過。
- 建立 public GitHub repo，`main` 為預設分支；設定 `atari-st`、`emulator`、
  `ai-tools`、`retrocomputing`、`golang`、`testing` 主題。
- 固定 SingleStepTests/m68000 commit 與授權，建立 NOP READY 規格、24-bit sparse bus、
  預取模型與 corpus loader；以 2,500 筆狀態／clock／bus trace 驗收第一條 CPU 指令。
- 修正外部 corpus 的 Docker 邊界：主機端解析與驗檔，容器內固定唯讀 `/corpus`，
  避免相對路徑在 Codex／Claude Code 工作目錄間漂移。
- 新增 MOVEQ READY 規格與實作；以 2,500 筆外部語料驗 Dn、符號延伸、CCR、預取及 bus。
- 依 Dungeon Master DM12EN 重建組語的實際使用情形，新增 SWAP、EXT.W、EXT.L；三份
  外部語料共 7,500 筆完整狀態、RAM、clock 與 bus transaction 全部通過，CPU 累計
  驗收 12,500 筆。
- 固定 NXP／Motorola 官方 M68000PRM 手冊雜湊，建立 Bcc／BRA 正常控制流規格；
  1,830 筆偶數目標語料全部通過，CPU 累計正常單步驗收 14,330 筆。另確認 670 筆
  奇數目標屬 address error，尚未實作例外時採失敗即關閉且不改狀態。

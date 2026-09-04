# 001 — 首版範圍

狀態：**READY**（使用者定案，2026-09-05）。

## 目標

提供可嵌入 Go 測試、亦可由 JSON Lines 程序控制的決定性 Atari ST／STF 無頭執行器，
讓 Codex、Claude Code 與其他自動化工具取得可重播、可量測的原版對拍證據。

## 納入

- Motorola 68000 與 ST／STF 遊戲路徑所需硬體。
- 指令／畫格推進、輸入、CPU／記憶體讀取、breakpoint、watchpoint、trace、快照、
  framebuffer 與音訊證據。
- Hatari 作外部交叉 oracle。
- 《Dungeon Master》作第一個真實遊戲 vertical slice，核心不得寫死遊戲位址或規則。

## 排除

- 第一階段不做 STE、TT、Falcon、68030、DSP 或 Videl。
- 不提供或下載 TOS ROM、遊戲磁碟與受保護素材。
- 不移植 Hatari／Steem 程式碼。
- 第一階段不以玩家向視窗、設定 GUI 或即時音訊播放為完成條件。

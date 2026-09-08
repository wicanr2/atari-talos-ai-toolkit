# 147 — `talos-jsonl/1` 的輸入命令與最小執行迴圈

狀態：**CONFORMED**。

## 範圍

到目前為止，機器只能從 Go 測試裡驅動；`talos-jsonl/1` 的 `key`、`mouse` 一直是
保留的 op 名，回 `not_implemented`。本切片把機器接進契約，讓外面送得進輸入、
跑得動指令、拿得到畫面的指紋。

實作的 op：`boot`、`reset`、`run_instructions`、`key`、`mouse`、`framebuffer`。
`capabilities` 跟著更新。

**不在本切片**：`run_frames`、`read_memory`／`write_memory`、`breakpoint`／
`watchpoint`、`snapshot`／`restore`、`trace`，以及把整張畫面的像素送出去
（`framebuffer` 只回指紋與幾何）。這些仍然明確回 `not_implemented`。

## 契約

1. **ROM 由行程啟動時決定**，不由請求指定：`TALOS_TOS_ROM` 指到 TOS／EmuTOS
   映像。沒有設就 `boot` 失敗——請求裡不接受檔案路徑，避免把讀檔權限交給對面。
2. `boot`：建立機器。已經開過要先 `reset`。
3. `reset`：丟掉目前的機器，回到未開機狀態。
4. `run_instructions`：`count` 條指令（上限一次 100,000,000）。**遇到匯流排例外
   就停下來，回 `bus_fault` 與訊息**——失敗即關閉在這一層是「回報」，不是吞掉。
   成功時回 `instructions`、`interrupts`、`clocks`。
5. `mouse`：`dx`、`dy`（各限 −128..127）、`left`、`right`。走規格 142／143 的
   相對滑鼠封包。
6. `key`：`scan_code`（`$01`–`$72`）、`pressed`。走規格 145 的 make／break。
7. `framebuffer`：回 `base`、`resolution`、`bytes`（32,000）與 32,000 個位元組的
   `sha256`。**不回像素**——那不是這一層該扛的資料量。
8. 機器還沒 `boot` 就送輸入或執行，回 `not_booted`。
9. 未知欄位一律拒絕（`DisallowUnknownFields` 沿用）。

## 驗收與停止線

- protocol 層：每個 op 的成功與失敗路徑、參數界限、未 boot 的保護、未知欄位。
- 端到端：`boot` → `run_instructions` 走到桌面 → `framebuffer` 的 `sha256`
  等於規格 140 釘的那個值 → `mouse` 移動 → 指紋改變 → 移回來 → 指紋復原。
- **不宣稱契約已完整**：上面列的那些 op 還是 `not_implemented`，
  `capabilities` 據實回報。

## CONFORMED 收據

- 2026-09-06：protocol 的 synthetic 與端到端都通過。`capabilities` 的
  `emulation_ready` 從 `false` 改成 `true`，`commands` 列出實際實作的那幾個。

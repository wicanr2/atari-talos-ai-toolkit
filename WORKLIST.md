# Atari Talos 工作清單

| 項目 | 狀態 | 完成條件 |
|---|---|---|
| M0 專案與控制契約 | **完成** | public repo、Docker 全測試、JSONL golden test |
| 68000 語料盤點 | **完成** | 固定來源、授權、版本、SHA-256 與測試載入器 |
| 68000 NOP 規格 | **CONFORMED** | 2,500 筆狀態、預取、clock 與 bus transaction 全同 |
| 68000 MOVEQ 規格 | **CONFORMED** | 2,500 筆暫存器、CCR、預取、clock 與 bus transaction 全同 |
| 其餘 68000 指令 | 進行中 | 每組 READY 後實作，逐組通過外部語料 |
| ST／STF memory map | 待辦 | TOS ROM 唯讀、RAM 邊界與 bus error 有規格及測試 |
| Hatari 外部 oracle | 待辦 | 同輸入 metadata、狀態與截圖收據可重跑 |

# 驗證矩陣

| 能力 | 內部測試 | 獨立 oracle | 現況 |
|---|---|---|---|
| JSON Lines 解碼與回覆關聯 | protocol／CLI golden test | 不適用 | 已建立 |
| 未知欄位與未知命令拒絕 | protocol／CLI test | 不適用 | 已建立 |
| 68000 NOP | 本地 pipeline／fail-closed 測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 MOVEQ | 本地 immediate／CCR 測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 SWAP／EXT.W／EXT.L | 本地暫存器／CCR 測試 | SingleStepTests 7,500 筆狀態＋bus trace | 通過 |
| 68000 Bcc／BRA 正常控制流 | condition／位移／pipeline 測試 | SingleStepTests 1,830 筆正常狀態＋bus trace | 通過 |
| 68000 address error | 固定 frame／模式／handler 測試 | Bcc 670 筆例外狀態＋bus trace | 通過 |
| 68000 BSR | stack／return PC／複合例外測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 RTS | stack pop／return 預取／複合例外測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 其餘 68000 指令 | 待建立 | SingleStepTests；TAS／TRAPV 暫不採信 | 進行中 |
| TOS 開機 | 待建立 | Hatari 同版本／同 ROM | 未開始 |
| 畫面 | 待建立 | Hatari 同幀原生 framebuffer | 未開始 |
| 輸入與時序 | 待建立 | Hatari 同事件與狀態點 | 未開始 |
| Dungeon Master | 待建立 | Hatari 正常入口同狀態路徑 | 未開始 |

# 043 — Motorola 68000 SUBA／CMPA

狀態：**CONFORMED**（2026-09-05）。

## 範圍與證據

本規格涵蓋 `SUBA.W/L <source>,An` 與 `CMPA.W/L <source>,An` 的全部合法 data source EA。
驗收輸入為 spec 003 固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5`：

| 語料 | 正常／來源 address error | SHA-256 |
|---|---:|---|
| `SUBA.w.json.bin` | 1,626／874 | `cbc43f9db2e44dd8d869345047b444773f73ed0af376723987b7faabca30d0cb` |
| `SUBA.l.json.bin` | 1,676／824 | `788a2d5fac9563260f7771456425f484e812887a45a77b82d33cea1e70df4022` |
| `CMPA.w.json.bin` | 1,664／836 | `f49365f9816aa621f2a00b3e1b9eb96038221b1504a4c20bf5b5a4ec61cdee2f` |
| `CMPA.l.json.bin` | 1,670／830 | `efd3c67626c60e4ec07e6174b8f98a8b302ba712310a041e3a516154524a40b2` |

DM12EN 重建組語集合雜湊見 spec 037；靜態盤點為 SUBA 8、CMPA 8 個使用點，
涵蓋 Dn／An／immediate source 與 word／long。此數量只用於實作優先序。

## typed 行為

1. word source 先符號延伸至 32 bits；long source 保留全部 32 bits。
2. SUBA 以目的 An 減 source，32-bit 回繞後寫回 An；完全不修改 CCR。
3. CMPA 以目的 An 減 source 計算 32-bit 比較旗標，不寫回 An；X 保留，NZVC 依
   32-bit subtraction 更新。
4. SUBA.W 正常 clocks 為 `8 + sourceCost`；SUBA.L memory source 為
   `6 + sourceCost`，register／immediate source 再加 2。CMPA.W/L 都為
   `6 + sourceCost`。
5. source EA、program／data FC、alignment、side effect、fault PC、14-byte vector-3
   frame 與 bus 次序沿用已 CONFORMED 的 typed reader 契約，逐筆以四份語料驗收。
6. CMPA opmode 判定須先於 CMPM mask；否則合法的 `CMPA.L An,An` 會被較寬鬆的
   CMPM mask 誤路由。

## 失敗模式與驗收

不合法 source EA、其他 SUB／CMP 變體、未實作 opcode 與 backend bus fault 必須回傳錯誤。
不改控制協定、素材、存檔或權利邊界。四份語料共 10,000 筆逐筆比較完整 state、RAM、
clock 與非 idle bus transaction；既有 200,000 筆不得回歸。
2026-09-05 驗收結果為 10,000／10,000 全數通過；全套外部單步語料累計
210,000 筆全數通過。

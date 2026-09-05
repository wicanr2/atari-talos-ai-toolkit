# 046 — Motorola 68000 MOVE to CCR／SR

狀態：**CONFORMED**（2026-09-05）。

## 範圍與證據

本規格涵蓋 MC68000 的 `MOVE.W <data EA>,CCR` 與 `MOVE.W <data EA>,SR`。
驗收輸入為 spec 003 固定的 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5`：

| 語料 | 分母與分類 | SHA-256 |
|---|---:|---|
| `MOVEtoCCR.json.bin` | 2,500；正常 1,504、source address error 996 | `603ffea2c9109daa2358288934337a52976636c59d008a75a5ba71d70b304729` |
| `MOVEtoSR.json.bin` | 2,500；supervisor 正常／fault 1,210、user privilege 1,290 | `cf7ce17aacfc6be5a9db9456bd7d7d70206c13e456691245011064e6c0a43d75` |

這是 TOS interrupt mask、supervisor mode 與 condition code 管理的必要 CPU 契約。
MOVE from SR 與立即數邏輯到 CCR／SR 另立規格。

## typed 行為

1. `MOVE to CCR` 在 user／supervisor mode 都合法；以 source bits 4–0 取代 CCR 的
   X／N／Z／V／C，SR bits 15–8 保留，其他未實作／保留 bits 維持清零。
2. `MOVE to SR` 只允許 supervisor mode；source 以 MC68000 implemented-bit mask
   `0xa71f` 載入完整 SR。user mode 不解析或讀取 source EA，直接以指令起始 PC 進
   privilege violation vector 8，固定 34 clocks。
3. 正常路徑兩者皆為 `12 + sourceCost` clocks。word source EA、alignment、side effect、
   fault PC、14-byte vector-3 frame 與 bus 次序沿用既有 CONFORMED typed reader。
4. `MOVE to SR` 載入新 S bit 後，最後 sequential prefetch 使用新 SR 決定的 user／
   supervisor program function code；來源讀取仍使用指令開始時的 data function code。
5. 狀態載入後先以新 program function code 重讀目前管線尾端 `PC-2`，再從 `PC`
   補入下一個 word；這筆 bus read 與後續 prefetch 重疊，不另增加總 clocks。

## 失敗模式與驗收

不合法 source EA、其他 `0x44Cx`／`0x46Cx` 編碼、未實作 opcode，以及 source／vector／
stack／prefetch backend bus fault 必須回傳錯誤。不改控制協定、素材、存檔或權利邊界。
兩份語料共 5,000 筆逐筆比較完整 state、RAM、clock 與非 idle bus transaction；
既有 217,500 筆不得回歸。
2026-09-05 驗收結果為 5,000／5,000 全數通過；全套外部單步語料累計
222,500 筆全數通過。

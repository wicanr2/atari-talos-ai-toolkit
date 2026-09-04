# 005 — Motorola 68000 暫存器轉換指令

狀態：**CONFORMED**（2026-09-05）。

## 證據與範圍

- 使用 spec 003 固定的 `SingleStepTests/m68000` commit 與 MAME microcoded core oracle。
- 語料檔為 `SWAP.json.bin`、`EXT.w.json.bin`、`EXT.l.json.bin`；實際筆數由語料標頭
  讀取，現行固定版本預期各 2,500 筆。
- 本規格只涵蓋資料暫存器直接模式，不延伸到 68020 的 `EXTB.L`。
- ReDMCSB DM12EN 的重建組語包含這三種指令，因此它們屬於 Dungeon Master 啟動路徑
  所需 CPU 覆蓋的一部分；出現次數只用來排優先序，不當作行為證據。

## 行為

1. `SWAP Dn` 交換 Dn 的高、低 16-bit 半字。
2. `EXT.W Dn` 將 Dn 低 8-bit 符號延伸至低 16-bit，高 16-bit 不變。
3. `EXT.L Dn` 將 Dn 低 16-bit 符號延伸至完整 32-bit。
4. 三者皆保留 X；依結果寬度更新 N、Z，並清除 V、C。
5. 其他資料暫存器、位址暫存器、USP、SSP 與 RAM 不變。
6. 預取、PC 與 4-clock program word read 與 spec 003 的 NOP 相同。

## 驗收

逐筆比較三份語料的完整 CPU 狀態、RAM、總 clock 與 bus transaction。三份語料全部
通過後才能標為 **CONFORMED**；外部語料未掛載時只允許跳過，不得冒稱已驗收。

2026-09-05 驗收結果：三份語料各 2,500 筆，共 7,500 筆全部通過；搭配 NOP 與
MOVEQ，現行 CPU 外部單步驗收累計 12,500 筆。

## 排除與停止線

- 不把 MAME 產生的語料描述成實機量測。
- 不由這三種指令推論例外、匯流排仲裁或其他定址模式。
- 不納入任何 TOS、遊戲映像或受保護原版資料。

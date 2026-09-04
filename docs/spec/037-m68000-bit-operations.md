# 037 — Motorola 68000 bit 操作

狀態：**CONFORMED**（2026-09-05）。

## 範圍與證據

本規格涵蓋 `BTST`、`BCHG`、`BCLR`、`BSET` 的 dynamic register bit number 與 immediate
bit number 兩種編碼。依 Motorola／NXP《M68000 Family Programmer's Reference Manual》
的 bit manipulation 定義，Dn destination 以 32-bit 操作，memory destination 只操作 byte；
只有 Z 反映操作前目標 bit，其他 SR bits 不變。

固定外部語料為 `SingleStepTests/m68000` commit
`64b253116a3de04aaac4346c43680960dc9b67e5`，由 MAME microcoded MC68000 core 產生：

| 語料 | 筆數 | SHA-256 |
|---|---:|---|
| `BTST.json.bin` | 2,500 | `085509a4341a447a3b9c83ffd73bda15402ddb409e30e875e507a290ac32b074` |
| `BCHG.json.bin` | 2,500 | `a5508b0dda62f57553e7d4d7a4cb6fd2cf1f16c27d8beb883a6978853ed3a285` |
| `BCLR.json.bin` | 2,500 | `2a5ce0b7e656a74c4bf4eb6cf0bf52cf00a3717b34d853cd563545047853902f` |
| `BSET.json.bin` | 2,500 | `90249455984d1b9fe03f85f4739a7535142653084b472fa8c17246aebdf57bab` |

DM12EN `OBJECT/ENGINE/FULL/DM12EN/*.S` 的排序逐檔 SHA-256 清單再雜湊為
`d7f92ecdec76c47071b598f53d270844490d097658bf6e7e982011abc866d975`；靜態盤點有
`BTST` 76、`BCHG` 0、`BCLR` 10、`BSET` 10 個使用點。此數量只決定實作優先序，
不等同執行期頻率。

## typed 行為

1. dynamic bit number 取 opcode 指定 Dn；immediate 型先消耗一個 extension word。
2. Dn destination 對 bit number 做 modulo 32；memory／immediate data destination 做
   modulo 8。先讀舊 bit：舊 bit 為 0 則設 Z，否則清 Z。
3. `BTST` 不改 operand；`BCHG` 翻轉、`BCLR` 清除、`BSET` 設定指定 bit。
4. memory 合法 EA：`(An)`、`(An)+`、`-(An)`、d16、brief-index、absolute word／long；
   `BTST` 另允許 PC-relative、PC-index 與 immediate data。修改型遇到這些 read-only
   source mode 必須失敗即關閉。
5. byte memory 次序是 EA extensions → operand read → final prefetch；修改型再做 operand
   write。A7 byte postincrement／predecrement 使用 2，其餘 An 使用 1；PC-relative read
   使用 program FC。
6. Dn clock 有資料相依微時序：dynamic `BTST/BCHG/BCLR/BSET` 在 bit 0–15 分別為
   6/6/8/6；修改型 bit 16–31 再加 2。immediate 型分別為 10/10/12/10，修改型
   bit 16–31 同樣再加 2。memory clock 由固定 EA extension 與 read／prefetch／write
   transaction 構成。

## 驗收與停止線

- 10,000 筆逐筆比較完整 D0–D7、A0–A7、USP／SSP、SR、PC、prefetch、RAM、總 clocks
  與非 idle bus transaction。
- 此語料沒有 address error 或 backend bus error；本規格不得拿正常 byte memory case
  宣稱 vector 2／3 已覆蓋。MOVEP、TAS、NBCD 及其他 opcode 另立規格。

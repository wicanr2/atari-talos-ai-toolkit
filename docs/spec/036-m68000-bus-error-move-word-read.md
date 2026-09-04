# 036 — MC68000 `MOVE.W` 來源讀取匯流排錯誤

狀態：**CONFORMED**（2026-09-05）。

## 範圍與停止線

本切片只處理 `MOVE.W <memory>,Dn` 的 word 來源讀取由 ST memory backend 回傳 typed
bus fault 時，進入 MC68000 vector 2。已由 Hatari 直接對拍的是 user data FC=1、
absolute-long 位址 `0x000000` 的 protected read。

目的端寫入、byte／long、instruction fetch、exception frame 寫入或 vector fetch 再次
fault，以及其他指令族均不在此切片；遇到這些情況仍失敗即關閉，不以本結果類推。

## Hatari oracle

- Hatari 2.4.1，映像 `sundog-atari-st-oracle:20260812`，immutable image ID
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`；
  `--machine st --memsize 1 --sound off --fast-boot true`。
- EmuTOS `emutos192us.img`，196,608 bytes，SHA-256
  `8fbbf8b44fc3e34281eaf8cda5265510e9af9ccda0e3e409111648060d244cfc`。
- probe PRG 為 54 bytes，SHA-256
  `1762cd11fe9db1b04dd3e89939f3b2c3f1e02a28e6dba84653f2f8ea081280e6`；先以 BIOS
  `Setexc(2, handler)` 安裝 vector 2，再執行 opcode `3039 0000 0000`
  (`MOVE.W $00000000,D0`)。handler 是兩個 `60fe` 自迴圈。
- debugger 在故障指令入口讀得 `FrameCycles=37496`，handler 首指令讀得
  `FrameCycles=37568`，差值為 **72 個 8 MHz CPU clocks**。
- Hatari 報告 `Bus Error reading at address $0`；handler 入口 ISP 為 `0x62be`，14-byte
  frame words 為 `3031 0000 0000 3039 0300 0001 3832`：依序是 SSW、fault address、
  opcode、原 SR、下一指令 PC。D0 保持 `0x00fc0370`。

## typed 行為

1. backend fault 以小介面傳遞 address、FC、read／write 與 size，CPU package 不依賴
   ST package；一般 backend error 不可被誤認為 vector 2。
2. 此切片只接受 read、size=2 的 typed fault。`MOVE.W` 已消耗的 extension transaction
   保留，之後加入 fault read、七次 frame word write、兩次 vector 2 read 與兩次 handler
   prefetch。
3. frame 排列與既有 vector 3 相同，但 vector fetch 改讀 `0x000008/0x00000a`。SSW 為
   `(opcode & 0xffe0) | FC | 0x0010`；saved PC 使用該 EA 已完成 extension consumption
   後的下一指令位置。
4. absolute-long source 的完成時間是 72 clocks；以既有 EA cost 表示為 `60 + cost`
   （absolute-long cost=12）。其他 EA 只受本機整合測試約束，不宣稱已由 Hatari 各自對拍。

## 驗收

- `internal/st` 整合測試以真實 `Memory` 觸發 low-memory protected word read，檢查 72 clocks、
  D0 不變、supervisor 切換、SSP 減 14、完整七個 frame words、vector 2 handler PC 與 prefetch。
- 原有 157,500 筆外部 CPU 單步語料必須全部保持通過；Hatari probe 是獨立 oracle 收據，
  不灌入該筆數。

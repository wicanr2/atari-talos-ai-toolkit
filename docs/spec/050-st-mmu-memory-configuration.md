# 050 — Atari ST MMU memory configuration register and 512 KiB-bank translation

狀態：**CONFORMED**。

## 範圍與證據

本切片建立 ST／STF `$FF8001` byte R/W 記憶體組態暫存器，以及本專案已支援的
512 KiB 與 1 MiB 實體 RAM 在 128 KiB／512 KiB／2 MiB 邏輯 bank 設定下的 STF
RAS／CAS 位址翻譯。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，第 27 頁。
  `$FF8001` 為 supervisor-only byte R/W，bits 3–2 選 bank 0，bits 1–0 選 bank 1；
  `00/01/10` 分別為 128 KiB／512 KiB／2 MiB，`11` 保留；表中高四位無定義。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false`。第一個 VBL 的 I/O trace 依序為：
  `read $FF8001=$00 @PC FC0052`、`write $0A @PC FC0188`、`write $05 @PC FC0218`。
  這證明 cold reset 初值、1 MiB 實體 RAM 最終組態，也證明 TOS 在檢測期會先以
  2 MiB + 2 MiB 邏輯設定存取較小的實體 bank。
  另一個 debugger 探針在同狀態寫入 `$FA` 後讀回 `$FA`；因此 latch
  保留完整 byte，只有 bank 大小解碼忽略高四位。
- **強證據（Hatari 2.3.1 可執行 oracle 實作）**：`stMemory.c:1949–2378`
  記錄 STF RAS／CAS 轉換、cold reset 歸零、bank 大小解碼與 512 KiB 實體 bank
  對三種邏輯大小的位元映射。本專案不複製 Hatari 程式碼，只依公開硬體
  契約重寫專案實際需要的兩種實體 RAM 拓撲。

## typed 行為

1. `NewMemory` 保留實體 RAM 容量：512 KiB 為 bank0=512 KiB、bank1 空；
   1 MiB 為兩個 512 KiB bank。MMU latch cold-reset 值為 0。
2. supervisor data／program FC=5／6 可對 `$FF8001` byte 讀寫；user 仍回
   `FaultProtected`。latch 寫入與讀回保留完整 byte，bank 解碼只使用低四位。
3. 組態碼每個 2-bit field 解為 128 KiB、512 KiB、2 MiB 或保留，邏輯
   RAM 窗口是兩 bank 之和。保留 field 不映射該 bank。
4. 512 KiB 實體 bank 的 STF 轉換：邏輯 2 MiB 移除 C9／R9，邏輯
   512 KiB 直接映射，邏輯 128 KiB 補入 C8=A17 與 R8=A9；最後限制在
   512 KiB 實體 bank。bank1 實體位址接在 bank0 後。
5. 邏輯 bank 指到未安裝的實體 bank，或位址超出兩個邏輯 bank 總和時，
   維持 typed `FaultUnmapped`。ROM、reset shadow、I/O 與權限路由先於 RAM 翻譯。
6. `Machine.Reset` 是 cold reset：先將 MMU latch 歸零，再由 ROM shadow 完成
   CPU reset。

## 驗收與停止線

- 兩種實體容量均驗 cold reset、supervisor byte R/W、高位保留但不參與解碼、user fault、
  `0x00`／`0x0A`／最終 `0x04` 或 `0x05` 組態。
- 1 MiB 機器驗 `0x0A` 下兩個 2 MiB 邏輯 bank 對兩個 512 KiB 實體 bank
  的 alias，再驗 `0x05` identity mapping；512 KiB 機器的 bank1 存取失敗。
- 以固定 EmuTOS ROM 從 reset 連續執行，必須越過先前第 3 條的
  `$FF8001` bus fault，並記錄新的第一失敗點；不因越過就宣稱 TOS 可開機。
- 128 KiB／2 MiB 實體晶片、STF 特殊 128 KiB+2 MiB void 區、STE／TT 位元交錯、
  DMA／Shifter 存取、warm reset 與 pin-level RAS／CAS 時序不在本切片。

## CONFORMED 收據

- 2026-09-05：Hatari I/O trace 直接確認 cold-reset `$00` 讀取，以及 EmuTOS
  後續 `$0A`、`$05` 寫入序列；單獨 `$FA` 寫讀探針確認高四位保留。
- 同 ROM 從 reset 執行至 `$FC0070` 的 `MOVEC D0,VBR` 探測邊界，Hatari 與
  Atari Talos 均為 7 條已完成指令、92 clocks、`SSP=$00001000`、
  `SR=$2704`、D0–D7／A0–A6 全零、prefetch=`$4E7B,$0801`。原第 3 條
  `$FF8001` reserved-I/O 失敗已消失。
- 內部測試涵蓋 512 KiB／1 MiB 實體 topology、cold reset、FC 權限、
  high-bit preservation、`$0A` alias、`$05` identity、空 bank fault 與 word atomicity；
  完整 Go 測試含 227,500 筆 CPU 外部語料、靜態檢查與建置全數通過。
- 新的第一開機停點是 MC68000 遇到 68010+ `MOVEC` 時尚未建立的
  illegal-instruction vector 4；這屬下一份規格，不是 MMU 差異。

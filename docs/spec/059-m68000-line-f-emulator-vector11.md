# 059 — MC68000 line-F emulator vector 11

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 MC68000 取到 `$Fxxx`（line 1111）opcode 時的 line-F emulator
例外 vector 11，直接解開 EmuTOS 1.3 在 `$FC00BE` 的 68030 PMMU 探測——
那是目前開機路徑的第一個停止點。

- **已確認（NXP 官方 MC68000 契約）**：《M68000 Family Programmer's
  Reference Manual》Appendix B 把 vector 11／offset `$02C` 定義為
  Line 1111 Emulator。PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
  **`$Fxxx` 在 MC68000 上一條合法指令都沒有**——它整段保留給 coprocessor
  介面，而 ST／STF 的 MC68000 沒有 coprocessor。所以把整個 `$F000`–`$FFFF`
  路由到 vector 11 不是「拿例外掩蓋未實作」，它就是這顆 CPU 的行為。
  這一點與 spec 051 的 vector 4 不同：那裡只有 `$4E7A`／`$4E7B` 兩個 opcode
  可以這樣路由，因為 `$4E7x` 的其餘編碼在 MC68000 上是合法指令。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false --trace cpu_disasm,cpu_regs`。

  ```text
  cpu video_cyc=   488 488@  0 488 : 00fc00ba 41fa 0096   lea.l (pc,$0096) == $00fc0152,a0
    A7 00000FE6   USP 00000000 ISP 00000FE6
    T=00 S=1 IMASK=7   Prefetch f010 (MMUOP030) 0800 (BTST)
  cpu video_cyc=   496 496@  0 496 : 00fc00be f010 0800   [ pmove (a0),tt0 ]
  cpu video_cyc=   532  24@  1 532 : 00fc00d4 21fc 00fc 00f2 0010   move.l ...
    A7 00000FE0   USP 00000000 ISP 00000FE0
    T=00 S=1 IMASK=7   Prefetch 21fc (MOVE) 00fc (ILLEGAL)
  ```

  也就是：opcode `$F010` 在 `$FC00BE`，例外花 **532 − 496 = 36 clocks**，
  SSP 由 `$0FE6` 減到 `$0FE0`（6-byte format-0 frame），SR 維持 `$2700`，
  handler 是 `$FC00D4`，handler 的前兩 words prefetch 成 `$21FC,$00FC`。

  handler 位址有一個獨立的佐證：同一顆 ROM 在 `$FC00B2` 執行
  `move.l #$00fc00d4,$002c`，把 `$FC00D4` 寫進位址 `$2C`——
  **`$2C` = 44 = 11 × 4**，正是 vector 11 的向量槽。

- Hatari 把 `$F010 $0800` 反組譯成 `pmove (a0),tt0`，那是 68030 的
  PMMU 指令。EmuTOS 用它探測有沒有 PMMU；在 MC68000 上得到 vector 11，
  handler 就是「沒有 PMMU」那條路。

## typed 行為

1. decoder 把 `opcode & 0xF000 == 0xF000` 全段路由到 vector 11。
   extension words 不解讀——例外在取到 opcode 時就決定了，`$F010` 後面的
   `$0800` 只是被 prefetch 進管線的下一個 word。
2. saved PC 為 `State.PC-4`（opcode 位址，不是 next PC），以原 SR 在
   supervisor stack 建立 6-byte format-0 frame；記憶體排列為 SR、
   PC high、PC low。與 spec 051 的 vector 4 同一條契約。
3. bus 寫入次序為 PC low、SR、PC high；再以 supervisor data FC=5 讀
   vector `$2C/$2E`，以 supervisor program FC=6 讀 handler 前兩 words。
4. 成功後 SSP 減 6，S 設 1、trace bit 清 0，其餘 SR 保留；PC 依 Atari
   Talos next-prefetch 契約為 handler+4，prefetch 是 handler 前兩 words。
5. 本例外固定 36 clocks。machine 只在成功時將 instruction count 加 1、
   clock count 加 36。

## 驗收與停止線

- synthetic bus 驗 supervisor／user 前態、SSP／USP bank、trace clear、
  saved opcode PC、frame 內容、vector／prefetch FC、bus 次序與 36 clocks，
  並涵蓋 `$F000`、`$F010`、`$FFFF` 三個端點以證明整段都走同一條路。
- 固定 EmuTOS ROM 必須從 reset 走到 `$FC00BE`，執行該指令後**這一步花
  36 clocks**、SSP 由 `$0FE6` 減到 `$0FE0`、SR 仍是 `$2700`、
  PC 落在 `$FC00D4+4`、prefetch 是 `$21FC,$00FC`、frame 三個 word 為
  `$2700,$00FC,$00BE`。
  比的是**這一步的 clock 差**而不是累積值：`MOVE.L #imm,(xxx).W` 的
  26／24 之爭還沒裁決（WORKLIST），它落在這一步之前，累積值因此會差 4。
  那個缺口是它自己的項目，本切片不靠放寬期望值吸收。
- 通過後繼續執行到新的第一失敗點；不因 vector 11 成功就宣稱 TOS 可開機。
- **line-A（`$Axxx`，vector 10）不在本切片**：它同樣是 MC68000 的
  unimplemented 段，但這顆 ROM 的開機路徑上沒有 line-A opcode，
  所以沒有端到端證據可收。要做要自己一片。
- 原生 `$4AFC` ILLEGAL、其他 illegal encoding、exception 進入期再
  bus fault／double fault、68020+ 的 coprocessor 協定不在本切片。

## CONFORMED 收據

- 2026-09-06：`$F000`、`$F010`、`$FFFF` 三個端點的 synthetic 測試通過
  supervisor／user、trace clear、SSP／USP bank、saved opcode PC、
  format-0 frame、全 bus 次序與 36 clocks；另一條測試證明三種
  extension word（`$0800`、`$0000`、`$FFFF`）走出完全相同的狀態。
- 固定 EmuTOS ROM 從 reset 走 18 條到 `$FC00BE`，前態 `SSP=$0FE6`、
  `SR=$2700`、prefetch=`$F010,$0800`；執行該指令**這一步花 36 clocks**，
  後態 `SSP=$0FE0`、`SR=$2700`、PC=`$FC00D4+4`、prefetch=`$21FC,$00FC`、
  frame=`$2700,$00FC,$00BE`。每一項都與 spec 上方引的 Hatari trace 相同。
- 開機路徑因此從**第 19 條推進到第 6851 條**，新的第一失敗點是
  `$FC0636` 的 `TST.B $FF860F`（DMA／FDC 那一段 I/O 還沒接）。
  **中間那 6832 條沒有對拍**：Hatari 的 1-VBL trace 只有 5685 條，
  而且兩邊在 I/O 上早就分岔了（Atari Talos 這一側大部分 I/O 仍是
  bus fault，EmuTOS 走的是「硬體不存在」那條分支）。所以這裡宣稱的
  只有「line-F 這一步全同」與「不再卡在第 19 條」，不是「開機路徑對拍」。

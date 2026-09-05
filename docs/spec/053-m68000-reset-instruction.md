# 053 — MC68000 `RESET` 指令與外部 reset line

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 supervisor 模式的 opcode `$4E70`：MC68000 保留 CPU register state、
對外部裝置送出 reset signal、推進 sequential prefetch，並以正確 clocks 完成。
它直接解開 EmuTOS 1.3 vector 2 handler 在 `$FC0088` 的 `RESET`。

- **已確認（NXP 官方 MC68000 契約）**：《M68000 Family Programmer's Reference
  Manual》將 `RESET` 定義為 privileged instruction；執行時 assert 外部 RESET
  signal，不重設執行該指令的 CPU internal state。PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false`。
- `RESET` 前的已驗證邊界是 10 條指令／220 clocks，handler opcode 位址
  `$FC0088`。Hatari 在下一 opcode `$FC008A` 停下時為 11 條完成指令對應的
  FrameCycles=352，因此 `RESET` 花 **132 clocks**。
- RESET 後 D0–D7／A0–A6 仍全零，`SSP=$0FEC`、`USP=0`、`SR=$2700`；
  prefetch=`$0CB9,$FA52`，下一個 opcode 位址是 `$FC008A`。

## typed 行為

1. opcode `$4E70` 只在 supervisor mode 執行；user mode 不送 external reset，
   而以既有 privilege-violation vector 8、saved opcode PC 與 34 clocks 契約處理。
2. CPU bus 可選擇實作 `M68KReset() error` typed hook。supervisor `RESET` 在
   sequential prefetch 前呼叫一次；hook error 原樣回傳，且不推進 prefetch。
   沒有外部裝置狀態的最小 bus 可不實作此 hook。
3. 成功的 external reset 不改 D／A／USP／SSP／SR；接著以 supervisor program
   FC=6 讀下一個 prefetch word，依既有 next-prefetch 契約移動 PC／queue。
4. 成功結果為 132 clocks；transaction 清單只記可觀察的 prefetch bus read，
   reset line 由 typed hook 表示，不偽裝成 memory transaction。
5. 目前 ST backend 收到 external reset 時將已建模的 MMU configuration latch
   回到 cold value `$00`；未實作的 GLUE、Shifter、MFP、PSG、ACIA、FDC 狀態
   留待各裝置 READY 規格接入同一 hook，不在本切片猜補。

## 驗收與停止線

- synthetic CPU 測試驗 supervisor 成功、hook 恰一次、132 clocks、register
  preservation、FC=6 prefetch；另驗 user privilege violation 不呼叫 hook。
- ST memory／machine 測試驗 external reset 將 MMU latch 歸零，但不清 RAM。
- 固定 EmuTOS ROM 從 reset 完成第 11 條指令後，必須是 352 clocks，state 與
  prefetch 對上 Hatari `$FC008A` 收據，然後找出新的第一失敗點。
- pin-level RESET pulse duration、尚未存在的周邊內部 reset 值、RESET 後的
  bus arbitration，以及整台機器的 power-on／使用者 reset button 不在本切片。

## CONFORMED 收據

- 2026-09-05：synthetic supervisor 測試確認 external reset hook 恰一次、
  registers 全保留、FC=6 sequential prefetch 與 132 clocks；user 測試確認
  不送 reset signal，改走 vector 8／34 clocks。
- ST memory 測試確認 external reset 將 MMU latch 歸零且不清 RAM。
- 固定 EmuTOS ROM 完成第 11 條指令後為 352 clocks、`PC=$FC008E`、
  prefetch=`$0CB9,$FA52`、`SSP=$0FEC`、`USP=0`、`SR=$2700`，D／A 全零，
  與 Hatari 在下一 opcode `$FC008A` 的收據一致。
- 後續可丟棄 probe 的新第一失敗點是 `$FC008A` `CMPI.L` 讀 `$FA0000`：
  Hatari 讀值 `$FFFFFFFF`，Atari Talos 尚將 cartridge window 視為 unmapped。

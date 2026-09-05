# 089 — ST YM2149 boot mixer／port A writes

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 開機時對 YM2149 的四筆 byte write：選 register 7、
寫 `$C0`，選 register 14、寫 `$07`。tone/noise/envelope、音訊合成、port I/O side effects、
byte mirror、word access與其他 registers 不在範圍，遇到時失敗即關閉。

- **已確認（Atari ST hardware map）**：YM2149 register select/read 位於 `$FF8800`，
  selected-register data write 位於 `$FF8802`；此專案沿用規格 035 固定的 supervisor
  I/O protection與 24-bit alias。
- **強證據（固定 Hatari 2.4.1 外部 oracle）**：`psg_write` trace 在 FrameCycles
  129,750／129,770／129,790／129,810 依序記錄 `$FF8800=$07`、`$FF8802=$C0`、
  `$FF8800=$0E`、`$FF8802=$07`，對應 register 7 與 14。reset trace在 machine reset
  與 MC68000 RESET 均清 YM state。
- **已確認（固定 EmuTOS ROM）**：SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；正常路徑第一筆
  未支援 write 的來源 D0=`$07`，位址 `$FF8800`。

## typed 行為與驗收

1. reset 將 selected register與 16 個 data bytes清零。
2. 固定序列只允許 select `$07` → data `$C0` → select `$0E` → data `$07`；每一步
   原子提交，最後 selected=`$0E`、R7=`$C0`、R14=`$07`。
3. 順序、register 或 value 不符，其他 PSG 位址／寬度、user access與未建模 read
   都失敗即關閉且不改 state。
4. synthetic test與固定 ROM已完成四筆 write，最後 selected/R7/R14=`$0E/$C0/$07`。
   正常路徑抵達 68,528 instructions、4 interrupts、968,510 clocks；下一 gate 是
   ACIA `$FFFC00`，完整 state 為 SSP=`$0F88`、SR=`$2304`、pipeline PC=`$FC51BC`、
   prefetch=`$FC00,$11FC`。Hatari 相鄰 write boundary 是 20 clocks；Talos 一般
   MOVE.B timed-I/O 尚未接線，故不宣稱四筆 cycle parity。完整 CPU corpus、全測試、
   vet 與 build通過後，本規格升 **CONFORMED**。

本切片不產生音訊，也不改 Dungeon Master 規則、畫面、存檔或權利邊界。

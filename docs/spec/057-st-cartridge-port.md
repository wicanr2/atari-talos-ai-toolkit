# 057 — ST cartridge port `$FA0000`–`$FBFFFF`

狀態：**CONFORMED**（2026-09-05）。

## 範圍與停止線

本切片只做一件事：**沒有插卡時**，讀取 cartridge port 位址區回什麼。

不在本切片：載入真實 cartridge 映像、寫入該區的行為、cartridge 的
diagnostic／application 兩種啟動模式，以及 EmuTOS 找到有效簽章之後會走的那條路。
那些都要有各自的證據才做——目前手上的 oracle 只涵蓋「沒插卡」這一種狀態。

## 平台規格證據

- Atari ST memory map：`$FA0000`–`$FBFFFF` 是 cartridge（ROM port）區，128 KiB。
  沒有卡匣時該區**沒有任何裝置驅動資料匯流排**，讀回的是匯流排的閒置狀態。
- EmuTOS 1.3 在 `$FC008A` 用 `CMPI.L #$FA52235F, $FA0000` 檢查卡匣簽章
  （`$FA52235F` 是 ST cartridge 的魔術字），不相等就在 `$FC0094` 分支跳過卡匣啟動。
  **這條指令直接讀該區，而且 EmuTOS 沒有為它預先安裝 bus error handler**——
  若該區會產生 bus error，開機就會在這裡走進 vector 2 而不是往下比對。

## Hatari oracle

用 `tools/hatari-oracle/trace.sh` 跑 EmuTOS 1.3 UK ROM
（`ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`），
Hatari 2.4.1、`--machine st --memsize 1 --fast-boot false`、未給任何 cartridge：

```text
$00fc0088	220	4e70                     reset
$00fc008a	352	0cb9 fa52 235f 00fa 0000 cmp.l #$fa52235f,$00fa0000 [ffffffff]
$00fc0094	380	660a                     bne.b #$0a == $00fc00a0 (T)
$00fc00a0	390	21fc 00fc 00b2 0010      move.l #$00fc00b2,$0010 [00fc0074]
$00fc00a8	416	203c 0000 0808           move.l #$00000808,d0
```

trace 把讀到的值印在方括號裡：**`[ffffffff]`**。所以沒插卡時讀 `$FA0000` 的 long 得
`$FFFFFFFF`，**而且沒有 bus error**——分支確實走到 `$FC00A0`，比對是正常完成的。

`$FC008A` 的 352 cycles 與 Atari Talos 在 spec 053 驗收到的 352 clocks 相同，
所以兩邊在進入這條指令時是同一個時間點，後面的差異才有意義。

## typed 行為

1. `$FA0000`–`$FBFFFF` 的讀取回 `$FF`，不產生 bus fault。這不是「假裝有記憶體」，
   而是空匯流排的實際樣子：沒有裝置拉低任何一條資料線。
2. function code 檢查不變——user mode 讀該區仍照 `validateAccess` 現行規則處理，
   本切片不動那條。
3. 寫入該區維持現行行為（fault）。沒有證據說真機在該區寫入會發生什麼，
   而 EmuTOS 開機路徑不寫它；憑空給它一個「寫入被吞掉」的行為是發明。

## 驗收與停止線

- synthetic：`$FA0000`、`$FA0002`、`$FBFFFE` 的 word 讀取全部得 `$FFFF`，不回錯；
  區界外的 `$F9FFFE` 與 `$FC0000` 維持原本行為（前者 unmapped、後者 TOS ROM）。
- 端到端：EmuTOS 1.3 UK ROM 自 reset 起，`$FC008A` 的 `CMPI.L` 執行成功並在
  `$FC0094` 取分支（**380 clocks**），到達 `$FC00A0` 時 **390 clocks**，
  與上面的 Hatari trace 相同。
- **驗收到 `$FC00A0` 為止。** 再下一條 `MOVE.L #imm,(xxx).W` 在 Hatari 是 26 cycles、
  在 Atari Talos 是 24，外部語料**沒有這個定址組合的案例**（只有 `(xxx).L` 一筆 28
  cycles），所以誰對還沒有第三方可以裁決。那是獨立的開放項，本切片不靠放寬期望值
  把它吸收掉——`$FC00A8` 因此不列入驗收。
- 既有語料與全套 Go 測試不得回歸。
- 通過後**不宣稱** TOS 可開機：只宣稱停止點再往後移。

## CONFORMED 收據

- 2026-09-05，synthetic：`$FA0000`、`$FA0002`、`$FBFFFE` 的 word 讀取都得 `$FFFF`；
  區界外的 `$F9FFFE` 仍 fault、`$FC0000` 仍是 TOS ROM——區間沒有被放寬。
- 2026-09-05，端到端：EmuTOS 1.3 UK ROM 的 `CMPI.L` 執行成功，`$FC0094` 取分支
  為 380 clocks、`$FC00A0` 為 390，與 Hatari trace 相同。
- 開機路徑從 12 條指令推進到 **18 條／494 clocks**，新的停止點是 `$FC00C2` 的
  `$F010`——line-F emulator 例外（vector 11），另立規格。
- 230,000 筆外部語料與全套 Go 測試在同一次執行中通過。

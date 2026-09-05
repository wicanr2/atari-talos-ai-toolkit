# 102 — Atari ST Ricoh `$FF860F` void byte write

狀態：**CONFORMED**。

## 範圍與證據

本切片延伸規格 058，只處理首版 `MACHINE_ST`／Ricoh chipset 對 `$FF860F` 的
supervisor byte write。這是普通 ST 的 void access，不是 Falcon mode control、
MegaSTE density register或 FDC／DMA 命令實作。

- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMem.c` SHA-256
  `ba67024fc35deeed202276e95844effbc014327f0893af71b85102601d2103cd`。
  `IoMem_SetVoidRegion()`（127–135）同時安裝 `IoMem_VoidRead` 與
  `IoMem_VoidWrite`；`IoMem_FixVoidAccessForST()`（144–148）將 `$FF860F`
  指向該區；`IoMem_VoidWrite()`（907–910）明確忽略 write。一般 ST register
  table `src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`，沒有
  `$FF860F` register。
- **強證據（固定 Hatari／EmuTOS 正常路徑）**：Hatari image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  `--machine st --memsize 1 --fast-boot false --timer-d false`。在 `$FC3788`
  執行 `MOVE.B D0,$860F`，D0 low byte=`$00`；Hatari I/O trace記為
  `IO write.b $FFFF860F = $00`，12 clocks後自然到 `$FC378C`，沒有例外。
  22 VBL trace SHA-256
  `4a480a9eba46dd5c6688a9f202a6d8dd740b095a8c6c5cacb6988b293b506c8b`；
  第一次寫入前總 clock 3,004,376，後一指令邊界為3,004,388。同一路徑之後再次
  寫 `$00`，證實不是一次性 probe state。
- **已確認（固定 Talos 正常路徑）**：規格 101 後第一個 gate 在289,520
  instructions、234 interrupts、2,982,748 clocks；pipeline PC=`$FC378E`、
  prefetch=`$860F,$0C2A`，bus fault位址 `$FF860F`，對應同一筆 byte write。

## typed 行為

1. `Memory.WriteByte($FF860F, value, supervisor data FC=5)`忽略任意 byte value，
   成功且不建立任何 register state；24-bit alias `$FFFF860F`同樣適用。
2. user data access仍由既有 I/O protection先行拒絕；word／long與相鄰位址仍維持
   失敗即關閉。read契約沿用規格058的`$FF`。
3. timed byte write只承擔既有 ST bus-slot wait；本切片不另加裝置 wait。
4. ColdReset／M68KReset不需新增狀態；這筆 write不得影響FDC、DMA、framebuffer、
   interrupt或其他機型設定。

## 驗收與停止線

- synthetic測試涵蓋任意值、24-bit alias、無狀態副作用、timed wait、user protection、
  相鄰位址與word access失敗即關閉。
- 固定ROM必須自然跨過第一次`$FC3788` write，完整核對instruction／interrupt／clock、
  CPU state與prefetch，再以有界步進定位下一個typed gate。
- 完整CPU corpus、固定ROM、全測試、vet與build通過後才升 **CONFORMED**。

固定ROM在289,521 instructions、234 interrupts、2,982,760 clocks完成write；完整
D/A、SSP=`$0F3C`、SR=`$2304`、pipeline PC=`$FC3790`與prefetch=`$0C2A,$0003`
均鎖入回歸測試。下一gate為289,549 instructions／2,983,072 clocks的PSG
`$FF8800=$0E`。完整240,000筆CPU corpus、固定ROM、全測試、vet與build通過。

本切片不聲稱 TOS 已開機，也不改 Dungeon Master 規則、資料、畫面、存檔或權利邊界。

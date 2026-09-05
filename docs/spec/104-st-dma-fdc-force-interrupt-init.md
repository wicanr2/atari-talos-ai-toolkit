# 104 — ST DMA mode與WD1772 force-interrupt初始化

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格103後固定EmuTOS第一組`$FF8606=$0080`、`$FF8604=$00D0` word
write，建立可供後續restore消費的DMA mode、FDC command/status與IRQ狀態。第二組
`$0080/$000B` restore、完成deadline、disk image、DMA transfer與其他register不在本切片。

- **強證據（固定Hatari 2.4.1原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
  檔頭register表（87–125）定義`$FF8606` bit7選FDC、bit4選sector count、bits2:1
  選WD1772 register；`$0080`因此選command/status。`FDC_DmaModeControl_WriteWord`
  （4322–4363）保存mode並增加4 wait clocks；`FDC_DiskController_WriteWord`
  （4085–4144）增加4 wait clocks並依mode把`$FF8604` low byte送到command register。
  `FDC_GetCmdType`（2043–2054）將`$D0`分類為Type IV；
  `FDC_TypeIV_ForceInterrupt`（3732–3767）在idle時切成Type-I status、motor on，
  condition low nibble=`0`故清IRQ，command完成時busy維持clear。
- **強證據（固定Hatari／EmuTOS正常路徑）**：Hatari image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  ROM SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  FDC trace SHA-256
  `cb3037b5faa0b9735ce97ec9899f7c834a541b3b17cc9c6fa67804f1808b7d07`：
  `$FC3720`在總clock 3,004,888寫mode `$0080`；`$FC3730`在3,005,326寫command
  `$D0`，隨即記錄Type IV、clear IRQ與complete。兩個word I/O均有4 wait-clock契約。
- **已確認（固定Talos正常路徑）**：第一個gate為289,565 instructions、234
  interrupts、2,983,240 clocks；fault位址`$FF8606`、value=`$0080`，pipeline
  PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. reset值為DMA mode=`$0000`、FDC command/status=`$00`、status type非Type-I、
   FDC IRQ inactive，且既有GPIP input bit5=`1`。
2. 只在規格103 stage3後接受supervisor word write`$FF8606=$0080`，保存mode並進
   init stage1；不啟動DMA或改FDC register。
3. stage1只接受`$FF8604=$00D0`。保存low-byte command=`$D0`，將status設為
   Type-I motor-on／busy-clear `$80`，IRQ保持inactive、GPIP bit5保持`1`，完成stage2。
4. 兩個timed word write在既有bus-slot wait之外各加4 device wait clocks；byte、long、
   user access、錯序、錯mode／command值與未建模read皆失敗即關閉且原子不變。
5. ColdReset／M68KReset清除新增DMA/FDC狀態。本切片不建立motor-stop timer；下一個
   restore會在它可能到期前取代command，且其deadline另立規格。

## 驗收與停止線

- synthetic測試涵蓋兩段狀態、routing、IRQ／GPIP、不合法順序與值、4 wait clocks及reset。
- 固定ROM自然完成`$D0`後鎖定CPU state／clock，再定位第二組mode／restore gate。
- 完整CPU corpus、固定ROM、全測試、vet與build通過後才升 **CONFORMED**。

固定ROM在289,612 instructions、234 interrupts、2,983,704 clocks完成`$D0`；完整
D/A、SSP=`$0F38`、SR=`$2310`、pipeline PC=`$FC373A`、prefetch=`$4E75,$2F0A`
與mode／command／status／IRQ／GPIP均鎖入回歸測試。下一gate在289,692
instructions／2,984,448 clocks，為第二次`$FF8606=$0080`；pipeline PC=`$FC3728`、
prefetch=`$8606,$2039`。完整240,000筆CPU corpus、固定ROM、全測試、vet與build通過。

本切片不修改Dungeon Master規則、資料、畫面、存檔或權利邊界；ROM與Hatari來源不入版控。

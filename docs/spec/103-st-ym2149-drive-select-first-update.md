# 103 — ST YM2149 port A 首次 drive-select 更新

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格102之後，固定EmuTOS透過YM2149 register 14讀改寫port A，將既有
`$07`改成`$05`的第一組序列。它不實作後續`$05→$03`、floppy motor／drive pin
side effect、音訊合成或其他PSG register。

- **已確認（既有平台契約）**：沿用規格089；`$FF8800`選擇／讀取目前register，
  `$FF8802`寫入所選register。固定boot狀態為selected=`$0E`、R7=`$C0`、R14=`$07`。
- **強證據（固定Hatari 2.4.1／EmuTOS正常路徑）**：image與ROM雜湊同規格102；
  CPU／I/O trace在總clock 3,004,710由`$FC36CA`執行`MOVE.B #$0E,(A0)`，寫
  `$FF8800=$0E`；3,004,726在`$FC36CE`讀回`$FF8800=$07`；經bit mask合成後，
  3,004,762在`$FC36DC`寫`$FF8802=$05`，下一邊界3,004,778。trace SHA-256
  `4a480a9eba46dd5c6688a9f202a6d8dd740b095a8c6c5cacb6988b293b506c8b`。
- **已確認（固定Talos正常路徑）**：規格102完成後，下一gate為289,549
  instructions、234 interrupts、2,983,072 clocks，fault位址`$FF8800`；pipeline
  PC=`$FC36D0`、prefetch=`$000E,$1010`，對應`$FC36CA`的immediate select write。

## typed行為

1. boot PSG狀態完成且未開始drive update時，只接受`$FF8800=$0E`同值select，
   進stage1；不改R7／R14。
2. stage1只接受supervisor byte read `$FF8800`，回目前R14=`$07`並進stage2。
3. stage2只接受`$FF8802=$05`，將R14原子更新為`$05`並完成stage3。
4. 錯序、錯值、user／word access與其他read維持失敗即關閉且不改stage／register。
   三筆timed access沿用既有PSG 4 wait-clock契約。
5. reset清除新增stage；本切片不把R14 bit意義或外部floppy狀態寫成已實作事實。

## 驗收與停止線

- synthetic測試涵蓋三段正常序列、read value、錯序／錯值原子拒絕、reset與timed wait。
- 固定ROM自然完成R14=`$05`，鎖定CPU state與clock，再有界定位下一typed gate。
- 完整CPU corpus、固定ROM、全測試、vet與build通過後才升 **CONFORMED**。

固定ROM在289,556 instructions、234 interrupts、2,983,132 clocks完成R14=`$05`；完整
D/A、SSP=`$0F34`、SR=`$2700`、pipeline PC=`$FC36E4`與prefetch=`$40C1,$46C2`
均鎖入回歸測試。下一gate為289,565 instructions／2,983,240 clocks的DMA mode/status
word `$FF8606=$0080`；pipeline PC=`$FC3728`、prefetch=`$8606,$2039`。完整240,000筆
CPU corpus、固定ROM、全測試、vet與build通過。

本切片不改 Dungeon Master 規則、資料、畫面、存檔或權利邊界。

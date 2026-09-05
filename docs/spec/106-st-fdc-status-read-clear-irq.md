# 106 — ST WD1772 restore後status read與IRQ清除

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格105後EmuTOS對`$FF8606`重寫DMA mode `$0080`，再從`$FF8604`
讀取Type-I status並清除FDC IRQ。後續track/data register、seek `$13`、DMA sector
transfer、磁片映像與一般化index／write-protect時序不在範圍。

- **已確認（EmuTOS VERSION_1_3正常原始碼）**：`bios/floppy.c` SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
  `flop_detect_drive()`（421–429）在restore完成後呼叫`get_fdc_reg(FDC_CS)`；
  `get_fdc_reg()`（1606–1611）先寫DMA control register selector、執行固定delay，
  再讀DMA data。`restore()`（1506–1510）同樣明寫status read resets IRQ。
- **強證據（固定Hatari 2.4.1原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
  `FDC_DmaModeControl_WriteWord`（4334–4361）保存完整mode並提供4 wait clocks；相同
  `$0080→$0080`不觸發bit8 DMA reset。`FDC_DiskControllerStatus_ReadWord`
  （4170–4323）在mode bits2:1=`00`時回Type-I status、提供4 wait clocks，並呼叫
  `FDC_ClearIRQ()`；後者（1932–1946）清IRQ source並把MFP GPIP5設回inactive high。
  無磁片但drive已啟用、head位於track 0時，動態status設TR00 bit2與WPRT bit6。
- **強證據（固定Hatari／EmuTOS正常路徑）**：Hatari image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  ROM SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`、
  trace SHA-256 `dd7077633bbcb2fd33d258e49cc904fbde42597b2323682d19b7381b8452ede9`。
  Restore完成並assert IRQ後，`$FC3888`寫`$FF8606=$0080`；`$FC3898`讀
  `$FF8604=$00E4`，trace在read前記錄clear IRQ。`$E4`為motor `$80`、無磁片所呈現
  WPRT `$40`、track-zero `$04`，busy已清。
- **已確認（固定Talos正常路徑）**：規格105後289,818 instructions、234 interrupts、
  2,985,802 clocks停在`$FF8606` word write；pipeline PC=`$FC3890`、
  prefetch=`$8606,$2039`。

## typed行為

1. FDC stage5只接受supervisor word write`$FF8606=$0080`；mode維持`$0080`、restore
   status與IRQ不變，進stage6並保留4 device wait clocks。此同值write不重設DMA。
2. stage6只接受supervisor word read`$FF8604`。在目前固定profile「drive 0 enabled、
   no disk、head track 0」回`$00E4`；讀取後FDC IRQ=false、GPIP5=`1`，進stage7。
   回傳值先取樣再清IRQ，因此本次read仍得到完成status。
3. status read保留4 device wait clocks與一般bus-slot wait。錯序、錯mode、user、
   byte／long、`$FF8606` read及其他未建模FDC register access失敗即關閉且原子不變。
4. reset清除新增stage；固定profile的no-disk WPRT只是本切片輸出，不外推插入磁片、
   index pulse或實體write-protect感測器。

## 驗收與停止線

- synthetic測試涵蓋同值mode、status取樣、read-clear IRQ／GPIP5、兩筆4 wait clocks、
  錯序／錯mode／user／byte與reset。
- 固定ROM須自然寫mode、讀得`$00E4`並清IRQ，鎖定CPU state／clock，再有界定位下一
  fail-closed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片只重現固定無磁片boot probe，不宣稱完整WD1772、DMA或可掛載遊戲磁片；不修改
Dungeon Master規則、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos在clock 2,986,242開始`$FF8604` timed word read，於289,865
  instructions、234 interrupts、2,986,256 clocks完成；D0=`$FFFF00E4`、
  PC=`$FC38A0`、prefetch=`$4E75,$2F0A`。read後IRQ=false、GPIP5=inactive high。
- 有界續跑到289,982 instructions、2,987,452 clocks；下一個失敗即關閉gate為
  `$FF8606` word write。依固定EmuTOS與Hatari正常路徑，值為`$0086`，用來選
  WD1772 data register；pipeline PC=`$FC3728`、prefetch=`$8606,$2039`。
- MC68000新增exact word-read phase；指令級測試鎖定operand read epoch與wait clocks。
  完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`及
  `go build ./...`通過。

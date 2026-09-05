# 105 — ST WD1772 restore完成期限與GPIP IRQ

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格104後第二組DMA mode `$0080`、WD1772 restore command `$0B`、
track-zero快速完成與FDC IRQ經MFP GPIP5供EmuTOS輪詢；並回填規格093留下的
ACIA receive IRQ line未接GPIP4缺口。後續status read清IRQ、其他FDC commands、
disk image、DMA FIFO／sector transfer與MFP GPIP edge interrupt不在範圍。

- **已確認（WD1772／Atari ST平台契約，固定Hatari可執行模型交叉驗證）**：DMA mode
  `$0080`選command/status register；`$0B`是Type-I restore，bit3禁止spin-up delay、
  verify bit清除、step-rate bits=`3`。Atari FDC IRQ接MFP GPIP5且active low；ACIA IRQ
  合併線接GPIP4且active low。
- **強證據（固定Hatari 2.4.1原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
  `FDC_TypeI_Restore`（3460–3476）置busy並以`90*8` FDC clocks進prepare；
  `FDC_UpdateRestoreCmd`（2215–2350）在drive已選、head track 0、verify off時直接置
  track-zero，再以`1*8` clocks進complete；`FDC_CmdCompleteCommon`（2070–2095）
  清busy並置IRQ。固定8 MHz FDC的總期限是728 FDC clocks，換算固定ST CPU clock為
  `floor(728*8021248/8000000)=729` CPU clocks；裝置只在CPU instruction boundary可見。
- **強證據（固定Hatari／EmuTOS正常路徑）**：Hatari image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  ROM SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  trace SHA-256 `f3bed6ec335b0f4d79b03278d593537b0752dd94e80f2da206ba66aed6a0c54e`：
  第二次mode在總clock 3,006,100附近寫`$0080`；`$FC3730`於3,006,538寫`$0B`，
  restore狀態為drive0／track0／spinup off／verify off。其後九次`$FC630C` GPIP read
  為`$B1`；3,007,288完成並assert IRQ，下一次read為`$91`。command write至實際
  callback相差750 clocks，與729-clock deadline加instruction-boundary可見延遲一致。
- **已確認（固定Talos正常路徑）**：接入word-write bus phase後，規格104後的
  第二次mode位於289,692 instructions、234 interrupts、2,984,448 clocks；位址
  `$FF8606`、value=`$0080`，pipeline
  PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. IKBD reset response RDR read清receive IRQ時，同步將combined ACIA IRQ input設為
   inactive high（GPIP4=`1`）；既有早期monitor probe `$A1`收據不受影響。
2. FDC init stage2只接受第二次supervisor word write`$FF8606=$0080`，mode不變、
   進stage3；stage3只接受`$FF8604=$000B`。
3. restore開始時command=`$0B`、status=`$81`（motor on＋busy）、Type-I=true、
   IRQ inactive、GPIP5=`1`，保存timed bus start clock並進stage4。
4. deadline=`start+floor(728*8021248/8000000)`；跨越時status=`$84`
   （motor on＋track zero、busy clear）、IRQ active、GPIP5=`0`、進stage5。多次輪詢
   只讀硬體線，不能自行倒數或觸發完成。
5. 兩個word write各保留4 device wait clocks。錯序／錯值、user、byte／long、未建模
   DMA/FDC read皆失敗即關閉且原子不變；reset清除pending、start clock與machine deadline。

## 驗收與停止線

- synthetic測試涵蓋ACIA GPIP4回線、第二組mode／restore、deadline前後status／GPIP、
  錯序／錯值、4 wait clocks與reset。
- 固定ROM必須自然完成九次inactive輪詢並讀到`$91`，鎖定CPU state／clock，再有界
  定位status-read清IRQ的下一gate。
- 完整CPU corpus、固定ROM、全測試、vet與build通過後才升 **CONFORMED**。

此期限是hardware-spec approximation加固定ROM同路徑驗證，不宣稱WD1772逐cycle內部
波形；本切片不修改Dungeon Master規則、資料、畫面、存檔或權利邊界。

## 驗收收據

- Talos以timed `MOVE.W 6(A7),$FFFF8604.W`在clock 2,984,902開始restore；deadline為
  2,985,631。EmuTOS自然完成九次GPIP5 inactive輪詢，於289,803 instructions、
  234 interrupts、2,985,654 clocks讀得`$91`；PC=`$FC6314`、prefetch=`$0801,$0005`。
- 有界續跑至289,818 instructions、2,985,802 clocks，下一個失敗即關閉gate是
  `$FF8606` word write，pipeline PC=`$FC3890`、prefetch=`$8606,$2039`。
- 完整240,000筆CPU corpus、固定ROM、全專案測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過。corpus另確認timed word-write接線未繞過來源或目的端奇數
  位址例外。

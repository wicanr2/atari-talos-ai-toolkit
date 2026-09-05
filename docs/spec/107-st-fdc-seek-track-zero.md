# 107 — ST WD1772 data register與same-track seek

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格106後第一顆drive probe：選WD1772 data register、寫目的track 0、
切回command register、執行seek `$13`、輪詢GPIP5、完成IRQ及Type-I status read-clear。
不同目的track、實際step、verify、index pulse、插入磁片、第二顆drive與DMA sector
transfer不在範圍。

- **已確認（EmuTOS VERSION_1_3正常原始碼）**：`bios/floppy.c` SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
  `flop_detect_drive()`（421–429）restore探測後保留track-zero結果；`seek()`
  （1497–1501）將目的track寫入FDC data register，呼叫seek後讀status清IRQ；
  `set_fdc_reg()`（1613–1618）明確採selector／delay／data順序。
- **強證據（固定Hatari 2.4.1原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`，`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
  mode `$86`選data register、`$80`選command/status。`FDC_TypeI_Seek`
  （3481–3504）置busy並排程`90*8` FDC clocks；`FDC_UpdateSeekCmd`
  （2362–2510）在motor已開、TR=DR=0、verify off時以零延遲穿過motor／same-track／
  verify states，再以`1*8` clocks進complete；總計728 FDC clocks，固定ST換算
  `floor(728*8021248/8000000)=729` CPU clocks。完成清busy並assert IRQ；status read
  契約沿用規格106。
- **強證據（固定Hatari／EmuTOS正常路徑）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；CPU＋FDC trace
  SHA-256 `476c4bf3cbfa8f1798cb6f510b5efcdba68b632f617c0363e7efddf7fcdb061f`。
  第一顆drive依序寫mode `$0086`、data `$0000`、mode `$0080`、command `$0013`。
  Hatari記錄motor already on、TR=DR=head track=`0`、verify off；command write總clock
  3,010,314，九次`$FC630C` GPIP read仍為`$B1`，3,011,064完成並assert IRQ，下一次
  `$FC630C` read為`$91`；其後status read回`$E4`並清IRQ。
- **已確認（固定Talos正常路徑）**：規格106後289,982 instructions、234 interrupts、
  2,987,452 clocks停在`$FF8606` word write；值依相同EmuTOS資料流為`$0086`，
  pipeline PC=`$FC3728`、prefetch=`$8606,$2039`。

## typed行為

1. stage7只接受supervisor word write`$FF8606=$0086`，mode更新並進stage8；stage8
   只接受`$FF8604=$0000`，保存data register=`$00`並進stage9。
2. stage9只接受`$FF8606=$0080`，進stage10；stage10只接受`$FF8604=$0013`。
   seek開始時command=`$13`、status=`$E5`（motor、no-disk WPRT、track-zero、busy）、
   Type-I=true、IRQ inactive、GPIP5 high，保存timed bus start clock並進stage11。
3. deadline=`start+729` CPU clocks；跨越時status=`$E4`、busy clear、IRQ active、GPIP5
   low並進stage12。GPIP輪詢不驅動倒數；固定正常路徑恰有九次inactive read，再讀`$91`。
4. stage12只接受`$FF8606=$0080`並進stage13；stage13只接受`$FF8604`word read，
   回`$00E4`後清IRQ、GPIP5 high並進stage14。
5. 四筆word write與status read各保留4 device wait clocks及一般bus-slot wait。
   錯序／錯值、user、byte／long與其他未建模FDC存取失敗即關閉且原子不變；reset
   清data、seek pending／clock／poll receipt及machine scheduler。

## 驗收與停止線

- synthetic測試涵蓋完整stage7–14、deadline前後、九次輪詢不驅動完成、read-clear、
  wait clocks、錯序／錯值／user與reset。
- 固定ROM須自然完成seek、九次inactive輪詢、active read及`$E4` status read，鎖定
  CPU state／clock，再有界定位下一gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

期限是hardware-spec approximation加固定正常路徑驗證，不宣稱不同track的step timing、
WD1772逐cycle波形或完整磁片行為；不修改Dungeon Master規則、素材、畫面與存檔。

## 驗收收據

- 固定Talos依序完成mode `$0086`、data `$0000`、mode `$0080`及seek `$0013`；
  seek timed command phase為2,988,614 clocks，deadline後恰有九次inactive GPIP read，
  再讀到active `$91`。
- status timed read phase為2,989,930；290,223 instructions、234 interrupts、
  2,989,944 clocks完成stage14。D0=`$FFFF00E4`、PC=`$FC38A0`、
  prefetch=`$4E75,$2F0A`，IRQ已清、GPIP5=inactive high。
- 有界續跑至290,296 instructions、2,990,830 clocks，下一gate是`$FF8800` byte
  write；固定EmuTOS／Hatari顯示這是選YM2149 register 14後將port A由drive 0切至
  drive 1的序列。pipeline PC=`$FC36D0`、prefetch=`$000E,$1010`。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`及
  `go build ./...`通過。

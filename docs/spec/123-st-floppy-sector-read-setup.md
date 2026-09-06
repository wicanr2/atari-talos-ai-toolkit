# 123 — ST floppy sector 1 DMA讀取設定

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格122已選定的drive 0／track 0，完整設定sector 1、DMA buffer／sector count，
並送出WD1772 Type-II read-sector `$80`。空磁碟機的command timeout、force interrupt、DMA／
FDC error decode與`flopunlk()`另立後續規格；本切片不把command提交冒充成功讀到資料。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1120–1165 flopio`每一sector
  先`set_fdc_reg(FDC_SR,sect)`、`set_dma_addr(iobufptr)`，讀取時呼叫
  `fdc_start_dma_read(1)`再`flopcmd(FDC_READ)`。`floppy.c:1613 set_fdc_reg`、其後
  `fdc_start_dma_read`證實register select／data與DMA write-bit toggle順序。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；680-VBL trace
  SHA-256 `4553ac450f460b968d7f008a48cf20d6acd19422ee85404a5885b61b5f0fdbe0`。
  VBL235在規格122後依序寫`$0084,$0001`（sector 1）、DMA address
  `$04,$10,$00`形成`$001004`、`$0190,$0090,$0001`（DMA count 1）、
  `$0080,$0080`（command register／read-sector command）。Hatari明記
  `type II read sector ... sector=1 ... dmasector=1 addr=0x1004`，隨後
  `start motor : no disk/drive`，沒有資料傳輸成功證據。
- **已確認（固定Talos入口與完成收據）**：入口為1,286,016 instructions／1,761
  interrupts／106,339,274 clocks，read stage 5、track 0、drive 0、port A `$25`、DMA
  mode `$0082`。固定ROM自然完成完整設定，在1,286,164 instructions／1,761 interrupts／
  106,340,824 clocks抵達read stage 15；`$80` command write clock為106,340,810，sector 1、
  DMA address `$001004`、sector count 1、兩次DMA reset與CPU完整狀態皆鎖定，且沒有bus gate。

## typed行為

1. read stage 5只接受`$FF8606=$0084`，接著`$FF8604=$0001`；保存sector register 1。
   錯register、值、寬度、user access與錯序均失敗即關閉。
2. 依low／middle／high順序接受DMA address `$04/$10/$00`並形成`$001004`；既有22-bit
   mask／even alignment契約不變，錯序不得推進read stage。
3. 依序接受DMA control `$0190→$0090`，每次direction bit toggle清DMA FIFO／sector
   count並保存本操作的reset count 2；再接受data `$0001`設定sector count 1。
4. 接受`$0080`選WD1772 command register，再接受data `$0080`提交single-sector Type-II
   read。保存command、sector、DMA address／count與提交clock；清FDC IRQ、GPIP5維持
   inactive high。不得修改RAM buffer或宣稱read成功。
5. command後的timeout／force-interrupt transaction仍失敗即關閉。cold reset清本流程的
   sector、address stage、reset count、command與clock收據。

## 驗收

- synthetic測試覆蓋完整10筆設定、錯序／錯值、DMA收據、無RAM資料寫入與reset。
- 固定ROM自然送出Type-II `$80`，鎖定完整CPU／clock狀態；完成點沒有未支援bus gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

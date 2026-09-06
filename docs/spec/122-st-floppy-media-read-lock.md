# 122 — ST floppy媒體確認讀取的track／drive設定

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理長時間空磁碟機輪詢後，EmuTOS進入`flop_mediach()`／`flopio()`讀取確認時，
`floplock(0)`為切換目前裝置所做的第一段設定：把WD1772 track register寫成drive 0保存的
track 0，再以YM2149 port A選取drive 0。後續sector register、DMA buffer、Type-II read
sector、timeout與error decode另立規格，不在此切片假造回應。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:455 flop_mediach`在write-protect
  latch成立且超過半秒時呼叫`flopio(...,RW_READ,dev,...)`；`floppy.c:1300 floplock`切換
  `cur_dev`時以`set_fdc_reg(FDC_TR,f->cur_track)`恢復該drive的track register。
  `floppy.c:1606 get_fdc_reg`／`:1613 set_fdc_reg`證實DMA control先選register、delay後才
  寫data。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；680-VBL
  `psg_all,fdc,video_vbl` trace SHA-256
  `4553ac450f460b968d7f008a48cf20d6acd19422ee85404a5885b61b5f0fdbe0`。
  首次確認讀取在VBL235依序為PC `$FC3720` `$FF8606=$0082`、PC `$FC3730`
  `$FF8604=$0000`，接著PC `$FC36CA/$FC36CE/$FC36DC`執行R14 select、讀`$23`、寫
  `$25`（drive `1→0`）；下一筆為`$FF8606=$0084`。
- **已確認（固定Talos入口）**：規格121之後為1,285,863 instructions／1,761 interrupts／
  106,337,672 clocks，PC `$FC3728`、prefetch `$8606,$2039`、bus fault `$FF8606`。
  對照相同原始PC與資料流，該word transaction為register selector `$0082`，不是D0中
  保存的track data `$0000`。

## typed行為

1. 僅在FDC初始化與ACSI掃描完成、media-check stage 8、至少一輪媒體檢查完成且沒有其他
   floppy read setup進行時，接受supervisor word `$FF8606=$0082`；保存DMA mode與read
   setup stage。錯值、user、錯寬度或重疊操作失敗即關閉。
2. 下一步只接受`$FF8604=$0000`，保存WD1772 track register 0與寫入clock；不得改寫
   sector register、command、DMA address、media-check count或既有FDC探測收據。
3. 再依序接受YM2149 `$FF8800=$0E`、讀R14舊值`$23`、`$FF8802=$25`；保存read target
   drive 0與stage。此序列不是`flopvbl()`，不得增加media-check count。
4. 完成後下一筆`$FF8606=$0084`仍失敗即關閉，作為sector-register後續規格入口。
   cold reset清read setup stage、track、drive與clock。

## 驗收

- synthetic測試覆蓋完整五筆transaction、每階段錯序／錯值、media count不變及reset。
- 固定ROM自然完成track／drive設定，鎖定CPU／clock狀態並在`$0084`邊界停止。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## CONFORMED收據

- Talos在1,285,863 instructions／106,337,672 clocks接受`$0082`，並於clock
  106,338,122提交track 0 data write；接著完成R14 `$23→$25`，media-check count保持73。
- 完成點為1,286,016 instructions／1,761 interrupts／106,339,274 clocks：read stage 5、
  track 0、drive 0、DMA mode `$0082`、port A `$25`。下一gate為相同PC `$FC3720`的
  `$FF8606=$0084` sector-register selector，與固定Hatari VBL235序列一致。
- synthetic完整序列、固定ROM、完整240,000筆CPU corpus、全測試、
  `go vet -stdmethods=false ./...`與`go build ./...`均通過。

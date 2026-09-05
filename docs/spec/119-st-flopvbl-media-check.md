# 119 — ST flopvbl drive-0 media check

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理固定EmuTOS在clock readback完成後，於VBL66執行`flopvbl()`的第一筆週期性
media-change檢查：暫時將YM2149 port A由`$23`改為`$25`以選drive 0，經ST DMA
mode `$0080`讀WD1772 status `$E4`，再把port A恢復`$23`（drive 1）。後續VBL的
drive-1檢查、真實diskette／write-protect變化、motor timeout與資料傳輸排除。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`。
  `bios/floppy.c:1350 flopvbl`每第八VBL選一顆drive，以`set_psg_porta()`暫存原port A、
  `get_fdc_reg(FDC_CS)`讀status，再恢復原值；`floppy.c:1417 set_psg_porta`只替換低三位。
  檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **已確認（固定EmuTOS原始碼）**：`bios/sound.c:115 ongibit`與共用PSG access說明
  證實control `$FF8800`選register 14、同址讀回selected data、`$FF8802`寫data；檔案
  SHA-256 `4b671a0f5af921dc793f750e3be8d3f7f4a6c01cbf9d501b151cbecfc1fd139c`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；67-VBL
  `psg_all,fdc,mfp_exception,mfp_read,mfp_write,video_vbl` trace SHA-256
  `22df5a57a16f3a49f24b3d24839b5a5eed89e47b7d60b3a49d9185c08a32e1d2`。
  VBL66依序為：PC `$FC36CA` select R14、讀`$23`、PC `$FC36DC`寫`$25`；
  `$FF8606=$0080`後PC `$FC3898`讀status `$E4`；再select R14、讀`$25`、寫回`$23`。
  Hatari明記drive `1→0→1`且side保持0。
- **已確認（固定Talos入口與勘誤）**：入口為1,005,202 instructions／521 interrupts／
  13,036,392 clocks、PC `$FC36D0`，bus fault address `$FF8800`。先前由D0=`$05`
  誤寫成`$FF8800=$05`；Hatari bus trace證實實際control write是`$0E`，D0不是該
  transaction value。本規格與`CONTEXT.md`明示訂正，不保留錯值作現行真相。

## typed行為

1. 僅在規格118 readback complete、既有PSG stage9、R7=`$C0`、R14=`$23`且FDC／ACSI
   初始化完成時，依序接受：control write `$0E`、control read回`$23`、data write
   `$25`。保存media-check stage，R14更新為`$25`；錯序／錯值原子拒絕。
2. 接受DMA control word `$0080`後，只允許WD1772 command/status word read；固定
   empty-drive profile回`$00E4`，清FDC IRQ並維持GPIP5 inactive high。不得重啟早期
   restore／seek probe stage或更改DMA address／sector count。
3. status讀完後依序接受control `$0E`、讀回`$25`、data `$23`；完成後R14=`$23`，
   drive identity恢復1並留下完整receipt。
4. user access、錯寬度、錯register、錯值、未完成readback或重複序列均失敗即關閉。
   cold reset清media-check stage與receipt。

## 驗收

- synthetic測試覆蓋完整六筆PSG＋兩筆FDC序列、每一步錯序／錯值、status副作用與reset。
- 固定ROM自然完成VBL66 drive `1→0→1`並鎖定CPU／clock邊界，再定位下一個真實gate。
- 固定ROM自然在1,005,296 instructions／521 interrupts／13,037,306 clocks完成；
  status取樣clock為13,036,978，CPU完整狀態由長路徑測試鎖定。
- 下一個真實gate為1,085,703 instructions／548 interrupts／13,927,048 clocks的
  `$FFFC02=$1C` byte write。固定Hatari 95-VBL trace確認VBL77再次發出讀時鐘命令，
  且仍回`$FC,$24,$03,$17,$00,$00,$00`；trace SHA-256
  `9c0f18cff88762f66e775b5241be37d342e6f3f545a8202810c6664ccb63b641`。這是後續
  規格，不在本切片猜補。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

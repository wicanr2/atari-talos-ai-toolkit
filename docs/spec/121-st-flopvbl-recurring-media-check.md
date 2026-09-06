# 121 — ST `flopvbl()`可重入雙磁碟機媒體檢查

狀態：**CONFORMED**。

## 範圍與停止線

本切片將規格119的單次drive-0媒體檢查收攏成可重入週期：每次依EmuTOS保存的輪替
身分檢查drive 0、drive 1、drive 0、drive 1；選取、讀WD1772 status後一律恢復呼叫前
YM2149 port A。固定無磁片profile不處理disk insertion、write-protect改變、motor timeout
或資料傳輸；下一個非此週期的裝置gate不在本規格猜補。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/floppy.c:1350 flopvbl`每第八VBL以
  靜態drive身分輪替檢查兩顆drive；`bios/floppy.c:1417 set_psg_porta`只替換port A低三位，
  status讀完後恢復原值。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **已確認（固定EmuTOS原始碼）**：`bios/sound.c:115 ongibit`及共用PSG access契約證實
  `$FF8800=$0E`選R14、同址讀selected data、`$FF8802`寫data；檔案SHA-256
  `4b671a0f5af921dc793f750e3be8d3f7f4a6c01cbf9d501b151cbecfc1fd139c`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；95-VBL trace
  SHA-256 `e8008be66d87fa348dd802202dee8d83e7c23bbe4909bb871efed0b41ca8ff91`。
  VBL66為`$23→$25`選drive 0後恢復`$23`；VBL74為`$23→$23`檢查drive 1；VBL82再選
  drive 0；VBL90再檢查drive 1。四輪皆寫DMA mode `$0080`、讀status `$E4`。
- **已確認（固定Talos入口）**：規格120完成後於1,120,640 instructions／568 interrupts／
  14,318,580 clocks，PC `$FC36D0`，bus fault `$FF8800`，prefetch為`$000E,$1010`。

## typed行為

1. 保存單調media-check count；count為偶數時下一輪選drive 0（port A `$25`），奇數時
   選drive 1（port A `$23`）。初始值0，故規格119首輪仍為drive 0。
2. stage 0或上輪完成stage 8時，依序接受select R14、讀原port `$23`、寫本輪target；
   再接受DMA control word `$0080`及WD1772 status word read `$E4`。錯序、錯值、錯寬度、
   user access或重疊週期均失敗即關閉。
3. status後依序select R14、讀本輪target、寫回原port `$23`；完成時count加一、保存本輪
   drive與status-read clock，stage停在8等待下一輪。FDC初始化、IRQ、GPIP與DMA位址不變。
4. cold reset清count、stage、本輪drive與收據；`flopVBLMediaComplete`只表示至少完成一輪，
   不再被誤解為週期永遠結束。

## 驗收

- synthetic連跑四輪，鎖定drive `0,1,0,1`、port target、status、count、錯序及reset。
- 固定ROM自然完成第二輪drive-1檢查，鎖定CPU／clock邊界並繼續定位下一gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`及
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## CONFORMED收據

- 第二輪選drive 1，port A全程為`$23`；status-read clock為14,319,166，完成點為
  1,120,734 instructions／568 interrupts／14,319,494 clocks。stage 8、count 2、
  last drive 1、FDC status `$E4`與原port均由固定ROM測試鎖定。
- 同一正常路徑未靠debug入口，繼續交替完成至第73輪；最後一輪為drive 0，status-read
  clock 105,344,570。下一個非輪詢gate為1,285,863 instructions／1,761 interrupts／
  106,337,672 clocks的`$FF8606` word write；此時media stage 8、DMA `$0080`、FDC `$E4`，
  因此不把新FDC transaction混入本規格。
- synthetic四輪鎖定`0,1,0,1`，固定ROM、完整240,000筆CPU corpus、全測試、
  `go vet -stdmethods=false ./...`與`go build ./...`均通過。

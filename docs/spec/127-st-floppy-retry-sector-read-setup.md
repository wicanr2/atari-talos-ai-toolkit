# 127 — ST floppy retry的sector 1 DMA讀取設定

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格126完成的drive 0同值重選，處理同一次`flopio()` retry重新送出的
sector 1、DMA buffer／sector count與WD1772 Type-II read-sector `$80`。這是第二次
transaction，必須保存獨立retry收據，不得覆寫第一次讀取設定的證據。後續無磁片timeout、
force interrupt與最終錯誤回傳另立規格；本切片不把command提交冒充成功讀到資料。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1081–1165 flopio`
  的retry仍從`select(dev,side)`進入每sector迴圈，依序呼叫
  `set_fdc_reg(FDC_SR,sect)`、`set_dma_addr(iobufptr)`、`fdc_start_dma_read(1)`與
  `flopcmd(FDC_READ)`。`floppy.c:1613`起的`set_fdc_reg`／DMA helper確認register
  selector、data及direction toggle順序。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`；
  EmuTOS官方`emutos-192k-1.3.zip`內`etos192uk.img` SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  330-VBL `fdc,psg_write,psg_read` trace SHA-256
  `af37cf3ecec5c31ea86650a6ef7f40ac8dcdd3b99bd60132f2fe7603c13849be`；VBL310
  在R14同值重選後依序寫`$0084,$0001`、DMA address `$04,$10,$00`、
  `$0190,$0090,$0001`及`$0080,$0080`。Hatari明記sector 1、track 0、side 0、
  drive 0、DMA count 1、address `$001004`，隨後為`no disk/drive`。
- **已確認（固定Talos入口）**：規格126後2,371,990 instructions／2,136 interrupts／
  118,369,170 clocks為read stage 27；下一gate在2,372,055 instructions／
  118,369,862 clocks寫`$FF8606=$0084`，PC=`$FC3728`、
  prefetch=`$8606,$2039`。

## typed行為

1. stage 27只接受`$FF8606=$0084`，stage 28只接受`$FF8604=$0001`，保存獨立
   retry sector 1收據；錯register、值、寬度、user access或錯序均失敗即關閉。
2. stage 29起依low／middle／high順序接受DMA address `$04/$10/$00`，形成
   `$001004`並保存retry address stage 3；不得更動第一次讀取的address-stage收據。
3. 依序接受DMA control `$0190→$0090`，保存retry reset count 2；接著接受data
   `$0001`，設定DMA sector count 1。
4. 接受`$0080`選WD1772 command register，再接受data `$0080`提交single-sector
   Type-II read。保存獨立retry command與精確bus clock；清FDC IRQ、GPIP5維持
   inactive high，status成為busy `$81`。不得修改RAM `$001004..$001203`或宣稱成功。
5. 每筆word FDC／DMA access沿用bus-slot wait加4 device clocks，DMA byte access沿用
   既有byte bus契約。cold reset清所有retry setup收據。

## 垂直鏈、驗收與權利邊界

- 本切片只解鎖EmuTOS正常無磁片retry路徑，不修改Dungeon Master規則、資料、素材、
  畫面、存檔或發行權利。
- synthetic測試覆蓋完整10筆設定、獨立收據、錯序／錯值、wait、RAM不變與reset。
- 固定ROM必須自然提交第二次Type-II `$80`，鎖定完整CPU／clock／bus收據；下一gate
  必須是此第二次command之後的真實正常路徑存取。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然完成第二次sector 1 Type-II `$80`提交：2,372,203 instructions／
  2,136 interrupts／118,371,412 clocks，精確command bus clock為118,371,398。
  retry sector、DMA address／stage、reset count、sector count與command分別為
  `$01`、`$001004/3`、2、1與`$80`；第一次transaction收據保持不變。
- command完成時FDC status為busy `$81`、Type-II、IRQ inactive，RAM
  `$001004..$001203`仍全零；完整D/A、SSP=`$687C`、SR=`$2310`、
  PC=`$FC373A`與prefetch=`$4E75,$2F0A`均鎖入固定ROM測試。
- 真正下一gate是3,456,990 instructions／2,511 interrupts／130,385,952 clocks的
  `$FF8606=$0080`第二次timeout selector；FDC仍為Type-II busy `$81`，沒有把等待
  或command提交外推成資料讀取成功。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

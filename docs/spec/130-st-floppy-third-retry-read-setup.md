# 130 — ST floppy第三次retry讀取設定

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格129完成的第二次dummy seek及其後既有status transaction，完整處理
第三次`flopio()` retry的drive 0同值重選、sector 1、DMA buffer／sector count與
WD1772 Type-II read-sector `$80`。第三次收據不得覆寫前兩次transaction。後續無磁片
timeout／force-interrupt、dummy seek及最終錯誤回傳另立規格；command提交不代表讀取成功。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1081–1165 flopio`
  每次操作先`select(dev,side)`，其後依序呼叫`set_fdc_reg(FDC_SR,sect)`、
  `set_dma_addr(iobufptr)`、`fdc_start_dma_read(1)`及`flopcmd(FDC_READ)`。
  `floppy.c:1417 set_psg_porta`與`1470–1476 select`令drive 0／side 0在既有高位
  `$20`下維持R14=`$25`。檔案SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **已確認（固定PSG access契約）**：EmuTOS `bios/sound.c:115 ongibit`及共用PSG
  access證實`$FF8800=$0E`選R14、同址讀selected data、`$FF8802`寫data；檔案
  SHA-256 `4b671a0f5af921dc793f750e3be8d3f7f4a6c01cbf9d501b151cbecfc1fd139c`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS官方1.3 192K UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；400-VBL
  PSG＋FDC trace SHA-256
  `97a7a5f348aec08ff36b2c5d23973f043818a652ac7d6c2039c993ce372a1d08`。
  VBL389依序記錄R14 select／read `$25`／write `$25`，再送`$0084/$0001`、DMA
  address `$04/$10/$00`、`$0190/$0090/$0001`及`$0080/$0080`；Hatari明記
  track 0、sector 1、side 0、drive 0、DMA count 1、address `$001004`，隨後
  `no disk/drive`，沒有成功傳輸證據。
- **已確認（固定Talos入口）**：規格129後3,516,206 instructions／2,528 interrupts／
  130,971,490 clocks抵達`$FF8800` supervisor byte write；PC=`$FC36D0`、
  prefetch=`$000E,$1010`，read stage 46、PSG stage 9、R14=`$25`、media count 73。

## typed行為

1. stage 46只接受`$FF8800=$0E`並進stage 47；stage 47只接受同址byte read、回
   `$25`並進stage 48；stage 48只接受`$FF8802=$25`，保存第三組drive-port／write
   epoch並進stage 49。R14及media count不變。
2. stage 49只接受`$FF8606=$0084`，stage 50只接受`$FF8604=$0001`，保存第三組
   sector 1收據並進stage 51。
3. stage 51起依low／middle／high順序接受DMA address `$04/$10/$00`，形成
   `$001004`並保存第三組address stage 3；不得覆寫前兩組address收據。
4. 依序接受DMA control `$0190→$0090`，保存第三組reset count 2；接受data
   `$0001`設定sector count 1，再接受`$0080`選command register及data `$0080`
   提交single-sector Type-II read。保存第三組command與bus clock，設busy `$81`、
   Type-II、IRQ inactive與GPIP5 high；不得修改RAM `$001004..$001203`。
5. PSG access各沿用4 device wait clocks；word FDC／DMA access沿用bus-slot wait加
   4 device clocks；DMA byte沿用既有契約。錯register、值、寬度、user access、錯序
   或重複存取均失敗即關閉且原子不變。
6. `MOVE.B Dn,d(An)`仍走未定時byte bus路徑，因此固定ROM PSG write收據保存
   instruction epoch，不冒稱精確bus phase；synthetic `WriteByteAt`仍保存精確clock。
   cold reset清第三組收據。

## 垂直鏈、驗收與權利邊界

- 本切片只解鎖EmuTOS正常無磁片第三次retry路徑，不修改Dungeon Master規則、資料、
  素材、畫面、存檔或發行權利。
- synthetic測試覆蓋完整13筆設定、獨立收據、錯序／錯值、wait、RAM不變、前兩組
  收據不變與reset。
- 固定ROM必須自然提交第三次Type-II `$80`，鎖定完整CPU／clock／receipt及下一gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收結果

- **已確認（Talos固定ROM）**：第三次R14同值寫回的instruction epoch為
  130,971,538 clocks；Type-II `$80`的精確bus clock為130,973,778。完整設定於
  3,516,426 instructions／2,528 interrupts／130,973,792 clocks抵達stage 59，
  drive `$25`、sector 1、DMA address stage 3、reset count 2與sector count 1皆由
  獨立第三組收據保存；FDC為Type-II busy `$81`、IRQ inactive，DMA buffer不變。
- **已確認（停止線）**：固定ROM再自然執行至4,600,388 instructions／2,903
  interrupts／142,979,288 clocks，下一個失敗即關閉閘門才是`$FF8606` word write；
  對應第三次無磁片timeout selector，另立規格，不納入本切片。
- synthetic完整覆蓋13筆順序、錯值拒絕、wait、三組收據隔離、RAM不變與cold reset。
  固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過，本規格升為 **CONFORMED**。

# 126 — ST floppy retry的drive 0同值重選

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格125完成的dummy seek／status read。EmuTOS準備下一次sector讀取時，再次
呼叫`select(0,0)`；YM2149 port A已是`$25`，因此序列會選R14、讀回`$25`、再寫回
同值`$25`。這是正常玩家路徑中的明確transaction，不是drive狀態改變，也不是週期性
`flopvbl()`媒體檢查。下一筆sector register `$0084`及完整第二次讀取另立規格。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c:1081–1120 flopio`
  在每次操作開始呼叫`select(dev,side)`，`floppy.c:1470–1476 select`將drive／side
  轉換值交給`set_psg_porta()`，`floppy.c:1417 set_psg_porta`只替換port A低三位。
  固定drive 0／side 0對應低三位`$05`；既有高位`$20`保留，結果為`$25`。檔案
  SHA-256 `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **已確認（固定PSG access契約）**：EmuTOS `bios/sound.c:115 ongibit`及共用PSG
  access證實`$FF8800`寫`$0E`選register 14、同址讀selected data、`$FF8802`寫data；
  檔案SHA-256 `4b671a0f5af921dc793f750e3be8d3f7f4a6c01cbf9d501b151cbecfc1fd139c`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；330-VBL
  `fdc,psg_write,psg_read` trace SHA-256
  `aa08a1b7743650950ad8e489659fc830efb9ca37383865777444a561b717c7dd`。
  VBL310的dummy seek status read後，PC `$FC36CA/$FC36CE/$FC36DC`依序select R14、
  read `$25`、write `$25`；接著PC `$FC3720`寫FDC control `$0084`。
- **已確認（固定Talos入口）**：規格125後2,371,983 instructions／2,136 interrupts／
  118,369,110 clocks抵達`$FF8800` supervisor byte write；PC=`$FC36D0`、
  prefetch=`$000E,$1010`，read stage 24、PSG stage 9、R14=`$25`。

## typed行為

1. read stage 24、PSG stage 9、selected register 14、R7=`$C0`、R14=`$25`且dummy seek
   已完成時，只接受`$FF8800=$0E`，保存register select並進stage 25。
2. stage 25只接受`$FF8800` byte read，回`$25`並進stage 26；不得更動R14。
3. stage 26只接受`$FF8802=$25`，保存同值drive-port收據與write instruction epoch並進stage 27。
   R14保持`$25`，drive 0／side 0不變，`flopVBLMediaChecks`不得增加。
4. 三筆PSG access各沿用4 device wait clocks。錯register、值、寬度、user access、
   錯序或重複存取均失敗即關閉且原子不變；cold reset清本輪收據。
5. 已知差異：目前`MOVE.B Dn,d(An)`仍走未定時byte bus路徑，故固定ROM收據保存
   instruction epoch而非精確bus phase；synthetic的`WriteByteAt`仍驗證可用時的精確
   access clock。不得把instruction epoch標成逐cycle bus parity。

## 垂直鏈、驗收與權利邊界

- 此切片只解鎖EmuTOS正常無磁片retry路徑，不修改Dungeon Master規則、資料、素材、
  畫面、存檔或發行權利。
- synthetic測試覆蓋三筆順序、同值不變、media count不變、錯序／錯值、wait與reset。
- 固定ROM必須自然完成同值重選，鎖定完整CPU／clock／receipt，下一gate必須是
  `$FF8606=$0084` sector selector。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

## 驗收收據

- 固定ROM自然完成R14 `$25→$25`；data write的instruction epoch為118,369,158，
  完成點為2,371,990 instructions／2,136 interrupts／118,369,170 clocks。R14保持
  `$25`且`flopVBLMediaChecks`保持73。
- 完成時D/A、SSP=`$6880`、SR=`$2700`、PC=`$FC36E4`與prefetch=`$40C1,$46C2`
  均鎖入固定ROM回歸測試。該epoch依已知差異不冒稱精確PSG bus phase。
- 下一gate為2,372,055 instructions／118,369,862 clocks的`$FF8606=$0084`；
  PC=`$FC3728`、prefetch=`$8606,$2039`，與固定Hatari trace一致。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

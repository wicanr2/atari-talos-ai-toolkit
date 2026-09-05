# 108 — ST YM2149 port A切換drive 1

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格107後固定EmuTOS再次讀改寫YM2149 register 14，將port A由`$05`
改為`$03`，使floppy選擇由drive 0切到drive 1。後續第二顆drive的FDC探測、side、
motor pin、其他PSG register與音訊合成不在範圍。

- **已確認（既有平台契約）**：沿用規格089與103；`$FF8800`選擇／讀取目前
  register，`$FF8802`寫入所選register。進入本切片時selected=`$0E`、R7=`$C0`、
  R14=`$05`，且drive-0 FDC探測已完成stage14。
- **強證據（固定Hatari 2.4.1／EmuTOS正常路徑）**：image與ROM雜湊同規格107；
  CPU／PSG trace SHA-256
  `013dbe3a2c7d20672e4ac3b862e9fa2cc285a583477a81caa0f0242ce5a4962b`。
  總clock 3,012,618由`$FC36CA`執行`MOVE.B #$0E,(A0)`；3,012,634於
  `$FC36CE`讀回`$05`；3,012,670於`$FC36DC`寫`$FF8802=$03`。Hatari隨後記錄
  port A由`$05→$03`、drive `0→1`、side維持0。
- **已確認（固定Talos正常路徑）**：規格107後290,296 instructions、234 interrupts、
  2,990,830 clocks停在`$FF8800` byte write；pipeline PC=`$FC36D0`、
  prefetch=`$000E,$1010`，對應上述immediate select write。

## typed行為

1. PSG drive stage3且FDC stage14時，只接受supervisor byte write`$FF8800=$0E`，
   進PSG stage4；不改R7／R14。
2. stage4只接受supervisor byte read`$FF8800`，回R14=`$05`並進stage5。
3. stage5只接受`$FF8802=$03`，將R14原子更新為`$03`並進stage6。
4. 錯序、錯值、user／word access與其他read失敗即關閉且不改stage／register。
   三筆timed byte access沿用PSG固定4 wait-clock契約；reset清stage與register。
5. 本切片只保存原版port值與選擇階段，不把未接線的side／motor或FDC內部狀態
   冒稱為已實作。

## 驗收與停止線

- synthetic測試涵蓋三段正常序列、read value、錯序／錯值原子拒絕、timed wait與reset。
- 固定ROM須自然完成R14=`$03`，鎖定CPU state／clock，再有界定位下一typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos自然完成`$0E→read $05→write $03`，於290,303 instructions、
  234 interrupts、2,990,890 clocks抵達PSG stage6；R14=`$03`、PC=`$FC36E4`、
  prefetch=`$40C1,$46C2`，D/A與stack均鎖入固定ROM測試。
- 有界續跑至290,312 instructions、2,990,998 clocks；下一gate為第二顆drive的
  `$FF8606=$0080`，pipeline PC=`$FC3728`、prefetch=`$8606,$2039`。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過。

# 118 — ST IKBD跨封包緩衝與clock readback

狀態：**CONFORMED**。

## 範圍與停止線

本切片承接規格117：set-clock最後一個`$00`已在shift register、TDRE已恢復且尚餘
10 ticks時，EmuTOS將第二個read-clock `$1C`緩衝進TDR。前一frame完成的同一serial
deadline把`$1C`移入shift register；其10-tick frame完成後，固定IKBD profile回
`$FC,$24,$03,$17,$00,$00,$00`，由既有MFP channel 6 handler交給`clockvec()`。
readback完成後的系統初始化、畫面、鍵盤／滑鼠與實時走秒排除。

## 證據

- **已確認（MC6850／既有typed契約）**：沿用規格091、092、115–117。TDRE表示TDR
  可收下一byte，不表示shift register已完成；當frame completion與pending TDR同時
  發生，先提交完成receipt，再在同一serial deadline載入pending byte。
- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/clock.c:976 igetregs`送`$1C`
  並等待`clockvec()`，`clock.c:994 isetregs`送`$1B + 6-byte`。檔案SHA-256
  `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；65-VBL
  `acia,ikbd_all,mfp_exception,mfp_read,mfp_write,video_vbl` trace SHA-256
  `eec38919b3cddea1047108e94bc9987017e5e9efdde8bb5dca62a76c122bd624`。
  VBL63 HBL280完成set-clock最後一byte並記
  `IKBD_Cmd_SetClock: 24 03 17 00 00 00`；同一串流已於HBL282載入`$1C`，HBL300完成
  並記`IKBD_Cmd_ReadClock: 24 03 17 00 00 00`。VBL64 HBL19起依序送
  `$FC/$24/$03/$17/$00/$00/$00`，每筆間隔20 HBL，均走vector table `$118`。
- **已確認（固定Talos入口）**：規格117停在881,554 instructions／473 interrupts／
  11,753,400 clocks；set-clock write count=7、complete count=6，shift byte=`$00`、
  ticks=10、TDRE=1、TDR仍為`$00`，guest正嘗試`$FFFC02=$1C`。

## typed行為

1. 僅在上述入口接受第二個`$1C` supervisor byte write；保存獨立readback-request
   receipt，寫TDR、設pending、清TDRE。錯值、重複、錯set-clock count、user／非byte
   access均原子拒絕。
2. 十個serial deadlines後先完成set-clock第七筆並設規格117 complete，再於同一
   deadline把pending `$1C`載入shift register；再過10 ticks提交獨立request-complete
   與clock receipt。
3. request完成後沿用規格116的16-tick首筆與10-tick後續期限，建立獨立readback
   queue `[FC,24,03,17,00,00,00]`。每筆仍須RDR空、置status `$83`、GPIP4 low、
   IPRB bit6、vector `$46`，讀取後清IRQ並保存順序與instruction-epoch receipt。
4. 第七筆讀完才設readback complete。cold reset清第二request、queue、deadline、
   delivery／read receipts；不得覆寫規格116第一輪收據。
5. 固定日期來自set-clock packet，不取host wall-clock，也不外推任意RTC或IKBD firmware。

## 驗收

- synthetic測試覆蓋跨封包buffer、同deadline complete/load順序、第二request 10 ticks、
  獨立response queue、RDR backpressure、MFP channel 6與cold reset。
- 固定ROM自然完成規格117第七frame、第二`$1C`與七筆readback；鎖定typed clocks與
  下一個真實失敗即關閉邊界。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過後才升 **CONFORMED**，並同步將規格117升CONFORMED。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## CONFORMED收據

- set-clock第七筆於11,763,550完成，並在同一deadline載入pending `$1C`；第二request
  frame於11,773,790完成。首筆readback deadline為11,790,174，後續依10,240 clocks
  間隔送達至11,851,614。
- guest七筆read instruction epochs為
  `11,790,462 / 11,800,706 / 11,810,938 / 11,821,178 / 11,831,430 /
  11,841,658 / 11,851,898`；內容為`[FC,24,03,17,00,00,00]`。後者仍是尚未遷移
  至timed read bus的指令起始epoch，不冒稱指令內精確bus phase。
- readback完成邊界為889,609 instructions／483 interrupts／11,851,910 clocks。
  後續正常執行至1,005,202 instructions／521 interrupts／13,036,392 clocks，下一gate
  是YM2149 `$FF8800=$05` register-select byte write。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`均通過。

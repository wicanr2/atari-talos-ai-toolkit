# 117 — ST IKBD set-clock傳送

狀態：**READY**。

## 範圍與停止線

本切片處理規格116的`clockvec()`完成後，固定EmuTOS 1.3 UK ROM因初始IKBD日期
month／day皆為零而呼叫`isetdt(DEFAULT_DATETIME)`，再經`isetregs()`與`ikbdws()`
傳送`$1B,$24,$03,$17,$00,$00,$00`七個bytes。每筆沿用MC6850單byte TDR、shift
register與10-tick 8N1 frame。IKBD更新內部時鐘後的第二筆`$1C` read-clock request、
其response、任意日期、host wall-clock同步與實時走秒皆排除。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/clock.c:994 isetregs`
  將command設為`$1B`並呼叫`ikbdws(6, &iclkbuf)`，`clock_init()`在IKBD month／day
  皆零時呼叫`isetdt(DEFAULT_DATETIME)`。`idosetdate()`／`idosettime()`將DOS日期時間
  轉為BCD。檔案SHA-256
  `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
- **已確認（MC6850既有平台契約）**：沿用規格091、092與115；TDRE=1時byte進
  TDR並清TDRE，下一個1024 CPU-clock serial deadline移入shift register並恢復
  TDRE；frame於10個serial ticks後完成。TDR可在前一frame傳送期間保存下一byte，
  但不得覆寫尚未移入shift register的pending byte。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；65-VBL
  `acia,ikbd_all,mfp_exception,mfp_read,mfp_write,video_vbl` trace SHA-256
  `231a85cc7407114764dfa9e33940da61bd0c0de8f46e054af1ecbf2e6d2994ed`。
  VBL63 HBL142先寫TDR `$1B`，HBL143起依序寫`$24/$03/$17/$00/$00/$00`；
  HBL280的`IKBD_Cmd_SetClock: 24 03 17 00 00 00`證實firmware完整消費封包，接著
  HBL282開始新的`$1C`，HBL300讀回`24 03 17 00 00 00`。
- **已確認（固定Talos入口）**：規格116後於874,900 instructions／471 interrupts／
  11,691,528 clocks停在`$FFFC02=$1B`；D2 low byte=`$1B`、PC=`$FC515A`、
  prefetch=`$FC02,$241F`，TDRE=1、無pending TX且clock response已完整讀取。

## typed行為

1. 只在上述固定入口接受依序七筆supervisor byte writes
   `[1B,24,03,17,00,00,00]`。每次接受都保存write receipt、寫TDR、設pending並清
   TDRE；錯值、錯序、重複、TDRE=0、已有pending、user或非byte access均原子拒絕。
2. ACIA另存正在傳送的shift byte，避免TDR在buffer下一byte後覆蓋前一frame身分。
   每個既有serial deadline先完成當前shift frame，再在同一deadline把pending TDR
   移入shift register、恢復TDRE並重新設10 ticks。
3. frame完成時必須與下一個預期receipt相同，依序保存七筆typed completion與clock；
   第七筆完成後才設set-clock complete。cold reset清write／completion receipts、
   shift byte與clocks。
4. 此為固定ROM profile，不把`$24/$03/$17`解讀成系統當下日期，也不讀host clock。
   serial scheduler仍是instruction-boundary hardware-spec approximation，不冒稱
   Hatari內部bit phase逐cycle parity。

## 驗收

- synthetic測試覆蓋七筆TDR buffering、shift身分、10-tick完成、順序／錯值／overrun
  拒絕、completion clocks及cold reset。
- 固定ROM自然送完整封包，鎖定七筆write與frame-completion receipts；以其後第二筆
  `$1C` write作IKBD已消費set-clock的下一gate，不在本規格猜補readback。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 實作中發現的下一閘門

固定ROM已自然接受七筆set-clock writes，前六筆frame completion clocks為
`11,702,110 / 11,712,350 / 11,722,590 / 11,732,830 / 11,743,070 /
11,753,310`。第七筆`$00`在shift register尚餘10 ticks、TDRE已恢復時，EmuTOS於
881,554 instructions／473 interrupts／11,753,400 clocks嘗試寫下一筆`$1C`。
這是跨packet TDR buffering的新行為，排除於本規格；在下一份READY規格接線並讓第七筆
自然完成以前，本規格維持 **READY**，不得升CONFORMED。

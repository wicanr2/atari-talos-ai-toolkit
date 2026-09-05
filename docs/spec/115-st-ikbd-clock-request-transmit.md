# 115 — ST IKBD clock request傳送

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理parallel-port strobe初始化後，EmuTOS `igetregs()`經既有`ikbdws()`
向IKBD傳送單一command byte `$1C`，以及MC6850 TDR進shift register、10個serial
ticks後完成8N1 frame。IKBD的`$FC + 6-byte` clock response、MFP channel 6 IRQ、
guest `clockvec()`、後續`$1B` set-clock packet與host wall-clock映射皆排除，另立規格。

## 證據

- **已確認（EmuTOS固定原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/clock.c:976 igetregs`
  清`iclk_ready`、設定`iclkbuf.cmd=$1C`並呼叫`ikbdws(0, &iclkbuf)`；檔案
  SHA-256 `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
- **已確認（MC6850既有平台契約）**：沿用規格091／092；configured且TDRE=1時
  data write進TDR並清TDRE，下一個1024 CPU-clock serial deadline把TDR移入shift
  register並恢復TDRE，10個serial ticks後完成該frame。
- **強證據（固定Hatari正常路徑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；70-VBL
  `acia,ikbd_all,video_vbl` trace SHA-256
  `9a308d18f840ffa0955eec3870b1136b3fc1cfeb62e16bfcf194ea5252a2d0de`。
  VBL62 HBL276先讀status `$02`，PC `$FC5154`寫`$FFFC02=$1C`；HBL295的
  `IKBD_Cmd_ReadClock`確認firmware收到`$1C`，之後才開始送response header `$FC`。
- **已確認（固定Talos入口）**：規格114後於867,320 instructions／462 interrupts／
  11,599,192 clocks停在`$FFFC02` byte write；D2 low byte=`$1C`、PC=`$FC515A`、
  prefetch=`$FC02,$241F`。此時IKBD ACIA已configured、TDRE=1、舊reset command
  frame與stale RDR read皆已完成。

## typed行為

1. 規格114完成、ACSI stage5、IKBD ACIA configured、TDRE=1、TDR仍記錄舊`$01`、
   shift counter=0且無pending write時，只接受supervisor byte write
   `$FFFC02=$1C`；更新TDR、設pending並清TDRE。
2. 下一個既有1024-clock serial deadline將`$1C`移入shift register，清pending、
   恢復TDRE並設10 ticks；每個後續deadline遞減，歸零時留下clock-request-complete
   typed receipt，且只提交一次。
3. 錯值、重複、錯前置狀態、user／word access均失敗即關閉且原子拒絕；cold reset
   清clock-request receipt。
4. 此切片沿用既有instruction-boundary serial scheduler近似，不宣稱Hatari裝置內部
   bit phase逐cycle parity，也不得在未實作response前假稱IKBD clock功能完成。

## 驗收

- synthetic測試覆蓋accept／reject、TDRE ownership、第一deadline與10-tick完成、
  receipt單次提交及cold reset。
- 固定ROM自然接受`$1C`並完成frame，鎖定Talos command write與frame完成邊界；
  response仍由下一規格接手，不把guest timeout路徑當成原版相同狀態。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收結果

- 固定Talos正常路徑於867,321 instructions／462 interrupts／11,599,204 clocks
  接受`$FFFC02=$1C`，TDR=`$1C`、pending=true、TDRE=0；PC=`$FC515C`、
  prefetch=`$241F,$245F`，完整D/A、stack與SR已鎖入測試。
- clock request在typed device clock 11,609,950完成10-tick frame；Talos於
  868,214 instructions／462 interrupts／11,609,966 clocks觀察completion receipt，
  TDRE=1、next ACIA clock=11,610,974。response仍明確排除，不沿guest timeout續跑。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`均通過。

# 116 — ST IKBD clock response與MFP channel 6

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理規格115的`$1C` request frame完成後，固定color-ST profile由IKBD回傳
`$FC,$00,$00,$00,$00,$00,$01`七個bytes；每筆經MC6850 RDRF／IRQ、MFP GPIP4
falling edge與channel 6 vector 70進入EmuTOS `_int_acia → _ikbdsys → ikbdraw`，最後
由`kbd_clock`呼叫`clockvec()`並令`iclk_ready=1`。後續`$1B + 6-byte` set-clock
packet、一般鍵盤／滑鼠、RX overrun與任意host wall-clock同步皆排除。

## 證據

- **已確認（EmuTOS固定原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`。
  `bios/clock.c:954 clockvec`複製六個payload bytes並設`iclk_ready=1`；
  `bios/clock.c:976 igetregs`等待該旗標，檔案SHA-256
  `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
  `bios/aciavecs.S:245 _int_acia`執行ACIA handler並以ISRB `$BF`清channel 6；
  `bios/aciavecs.S:382 _ikbdsys`讀status／RDR，`ikbdraw`由header `$FC`選action 4、
  收六個payload後由`kbd_clock`呼叫`clockvec`。檔案SHA-256
  `34ead2e8194760ea45039bae1bb48b6207bb9db66fbd11d973f1fb1188968f30`。
- **已確認（MC6850／MC68901既有平台契約）**：沿用規格093與096；RDR到達置
  RDRF與ACIA IRQ、active-low ACIA IRQ接ST MFP GPIP4／channel 6，VR base `$40`
  形成vector `$46`，vector table address=`$118`。讀RDR清RDRF／ACIA IRQ，guest
  software-EOI寫ISRB `$BF`清channel 6 in-service。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；64-VBL
  `acia,ikbd_all,mfp_exception,mfp_read,mfp_write,video_vbl` trace SHA-256
  `1acf376c48155a3587737535132a4c07620f43bae882237ec5fc7f108cb39b1d`。
  trace先記`IKBD_Cmd_ReadClock: 00 00 00 00 00 01`，再於VBL63 HBL
  12/32/52/72/92/112/132依序完成RDR `$FC/$00/$00/$00/$00/$00/$01`；每筆均
  `irq_new=1 → MFP iack vector address $118 → status $83 → read RDR → irq_new=0 →
  GPIP $B1 → ISRB $BF`。首筆相對request frame完成約16 serial ticks，後續間隔
  10 ticks；此為固定profile收據，不外推任意IKBD firmware。
- **已確認（固定Talos入口）**：規格115的request frame typed完成clock為
  11,609,950，觀察邊界868,214 instructions／462 interrupts／11,609,966 clocks，
  TDRE=1且尚無RDRF。

## typed行為

1. `$1C` frame完成時建立固定response queue
   `[FC,00,00,00,00,00,01]`；首筆deadline=`completion + 16*1024` CPU clocks，
   後續每筆deadline增加`10*1024` clocks。
2. deadline到達且RDR空時，只提交queue head：寫RDR、status置`$83`、GPIP4由high
   拉low並置IPRB bit6。RDR未清時不得覆寫或跳byte；同一筆維持pending。
3. MFP B-bank仲裁加入channel 6，優先於channel 5／4；IERB／IMRB bit6均enabled且
   無較高in-service時，以level 6、vector `$46`進handler，acknowledge將IPRB bit6
   轉成software-EOI ISRB bit6。
4. configured supervisor byte read RDR依序回queue bytes，清status bit7／0並使
   GPIP4回high；不得套用規格093只屬於`$F1` reset response的stale-RDR allowance。
   `_int_acia`先呼叫`aciavecs.S:295 _midisys`讀configured MIDI ACIA status；固定
   無資料狀態為`$02`，故不讀MIDI data。guest寫ISRB `$BF`後才能完成該次IRQ。
5. 七筆read receipts必須保持順序且完整；第七筆讀完後response complete，但
   `clockvec()`是否已執行由正常ROM後續進入`$1B` set-clock write作consumer證據。
6. cold reset清queue、index、deadlines、read receipts、channel 6 pending／in-service
   與response completion。錯序、錯來源或overrun狀態失敗即關閉。

## 驗收

- synthetic測試覆蓋七筆queue、首筆／後續deadline、RDR backpressure、GPIP edge、
  channel 6優先序與software EOI、RDR read receipts、`$F1` stale行為隔離及reset。
- 固定ROM自然接受七次vector 70，收齊payload並離開`igetregs()`；鎖定各delivery／
  read clock、完成CPU state／clock，再有界定位第一筆`$1B` set-clock gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## CONFORMED收據

- 固定ROM七筆delivery clocks為
  `11,626,334 / 11,636,574 / 11,646,814 / 11,657,054 / 11,667,294 /
  11,677,534 / 11,687,774`；guest read instruction epochs為
  `11,626,630 / 11,636,858 / 11,647,106 / 11,657,338 / 11,667,578 /
  11,677,826 / 11,688,058`。後者是尚未遷移至timed read bus的`MOVE.B`指令起始
  epoch，不冒稱為指令內精確bus phase。
- 完成邊界為874,579 instructions／471 interrupts／11,688,070 clocks；收件內容
  `[FC,00,00,00,00,00,01]`。隨後於874,900 instructions／11,691,528 clocks抵達
  `$FFFC02=$1B` write，證實`clockvec()`已完成且`igetregs()`進入set-clock路徑。
- 固定ROM、全測試、`go vet -stdmethods=false ./...`、`go build ./...`與完整
  240,000筆MC68000外部單步語料均通過。

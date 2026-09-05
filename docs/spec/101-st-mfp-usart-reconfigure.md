# 101 — ST MFP USART 第二次設定與 baud Timer D 重啟

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格100停止system Timer D後，固定EmuTOS第二次`rsconf`讀取既有USART
狀態、以control 1重啟baud Timer D，並重寫相同UCR／RSR／TSR／SCR值。實際serial
shift、UDR、GPIO handshake與USART IRQ不在範圍。

- **已確認（MC68901一手規格）**：TSR bit7是transmit buffer empty唯讀狀態；bit0是
  transmitter enable。Timer D control1是delay mode ÷4，data2因此每期8 MFP clocks；
  USART可使用Timer D輸出作baud clock，這不等於啟用Timer D channel 4 IRQ。
- **已確認（固定EmuTOS 1.3 source）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/serport.c:223 rsconf_mfp`
  先保存舊UCR／RSR／TSR，再由baud table呼叫`setup_timer(...,3,...)`，依參數寫
  UCR／RSR／TSR／SCR。
- **強證據（固定Hatari 2.4.1／EmuTOS ROM oracle）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。後段trace
  依序讀UCR=`$88`、RSR=`$01`、TSR=`$81`；TCDCR `$50→$50`、TDDR=`$02`、
  TCDCR=`$51`（`data=2 ctrl=1 timer_cyc=8`）；再於`$FC6B38/$FC6B40/$FC6B48/
  $FC6B50`寫`$88/$01/$01/$00`。
- **已確認（固定Talos正常路徑）**：規格100後第一個gate在289,332 instructions、
  234 interrupts、2,979,596 clocks，fault位址UCR `$FFFA29`，faulting opcode
  `$FC6B38`，不是D1=`$51`本身。

## typed行為

1. 已知TSR register保存可寫control bits；在未送出UDR且buffer空時，byte read回
   `mfpTSR | $80`。因此軟體reset值讀`$80`，enabled值讀`$81`；write仍只保存bit0。
2. 只有Timer D stop stage7、TCDCR=`$50`、TDDR/main=`$00`時接受TCDCR同值`$50`
   作reconfigure stage1；接著TDDR=`$02`為stage2，TCDCR=`$51`為stage3並設定
   Timer D running。
3. control1 baud Timer D不建立規格098的channel4 timeout／pending scheduler；只有
   TCDCR low bits=`2`的system Timer D可建立該scheduler。baud output本身留待serial
   data路徑需要時另立規格。
4. stage3後依序接受UCR=`$88`、RSR=`$01`、TSR=`$01`、SCR=`$00`，完成stage7。
   只開放此正常序列的同值writes；錯序／錯值失敗即關閉且不改stage。
5. Timer C、MFP channel enable／pending／in-service與vector均不變。

## 驗收與停止線

- synthetic測試涵蓋TSR `$80/$81` read、七段restart/rewrite、錯序拒絕，以及baud
  Timer D不啟動system IRQ scheduler。
- 固定ROM必須自然完成至SCR write並抵達下一個typed gate；不得直接改guest狀態。
- 完整CPU corpus、固定ROM、全測試、vet與build通過後才升CONFORMED。

固定 ROM 在 289,342 instructions、234 interrupts、2,979,680 clocks 自然完成 SCR
write；pipeline PC=`$FC6B58`、prefetch=`$2002,$4CDF`。Timer D control1保持 running，
但 system Timer D scheduler／deadline 均未啟動。後續有界探測於289,520 instructions、
2,982,748 clocks抵達DMA／FDC位址`$FF860F`的byte write gate；該裝置另立切片。
完整240,000筆CPU corpus、固定ROM、全測試、vet與build通過。

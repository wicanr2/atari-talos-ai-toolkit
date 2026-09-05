# 100 — ST MFP Timer D 正常停止與 channel 4 清除

狀態：**CONFORMED**。

## 範圍與證據

本切片處理固定 EmuTOS 在後段初始化停止規格 097／098 system Timer D、替換 vector，
再以共用 `mfpint(channel 4)` 清除 channel state 的完整序列。後續 USART／floppy 初始化、
Timer D current-counter capture與重新啟動不在範圍。

- **已確認（MC68901 一手規格）**：IERB／IMRB bit 4 分別控制 Timer D interrupt enable
  與 mask；TCDCR low three bits=`0` 停止 Timer D。IPRB／ISRB 是 write-zero-to-clear。
- **強證據（固定 Hatari 2.4.1／EmuTOS 1.3 UK oracle）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。固定 trace：
  `$FC7862 BCLR #4,(A0)` 令 IERB `$70→$60`；`$FC786A` 令 IMRB `$70→$60`；
  `$FC7872 AND.B #$F0,(A0)` 令 TCDCR `$52→$50` 並停止 Timer D。之後
  `$FC7878` 將 A1=`$FC03EA` 寫 vector table `$110`，再由 `$FC6194–$FC61AC`
  依序寫 IMRB=`$60`、IERB=`$60`、IPRB=`$EF`、ISRB=`$EF`。
- **已確認（固定 Talos 正常路徑）**：規格 099 後第一個 fault 在 289,230 instructions、
  234 interrupts、2,978,434 clocks；pipeline PC=`$FC786A`、prefetch=`$41F8,$FA15`，
  faulting opcode 是 `$FC7862 BCLR #4,(A0)`，不是 D0=`$A7` 的資料寫入。

## typed 行為

1. 只有 Timer D system stage 8、IERB/IMRB=`$70`、TCDCR=`$52`、channel 4 無 pending／
   in-service 時接受 IERB `$70→$60`，建立 stop stage 1。
2. stage 1 才接受 IMRB `$70→$60`；stage 2 才接受 TCDCR `$52→$50`，清
   `mfpTimerDStart` 並建立 stage 3。machine 觀察 transition 後清 Timer D scheduler 的
   started／period／deadline，之後不得再產生 bit 4 pending。
3. vector table `$110` 是一般 RAM write，沿用 CPU／memory 契約；本 MFP stage 不攔截它。
4. stage 3 後依序只接受 IMRB=`$60`、IERB=`$60`、IPRB=`$EF`、ISRB=`$EF`，完成 stage 7。
   已證實 IMRB 同值寫入即使 Timer C bit 5 正 pending 仍須接受並原樣保留該 pending；
   此例外不允許 mask 值改變，也不放寬其他 pending 下的 IMRB write。
   跳步、錯值、仍 pending／in-service 或其他 active-timer stop 均失敗即關閉且原子不變。
5. Timer C control high nibble、scheduler與channel 5 state全程不變。停止瞬間的Timer D
   current counter尚未建模；只要guest未讀TDDR，不以固定0冒稱真實capture。

## 驗收與停止線

- synthetic 測試涵蓋七段 sequence、錯序拒絕、Timer C 保留與 scheduler stop 後不再 pending。
- 固定 ROM 必須自然執行 `$FC7862/$FC786A/$FC7872`、更新 `$110=$FC03EA`，再由 guest
  完成 `$EF` clear；抵達下一個真正 gate 才升 CONFORMED。
- 完整 CPU corpus、固定 ROM、全測試、vet 與 build 通過後才升 **CONFORMED**。

## 驗收收據

- 固定 ROM 在 289,256 instructions、234 interrupts、2,978,730 clocks 完成 guest
  `$EF` clear；pipeline PC=`$FC61B4`、prefetch=`$4E75,$302F`。IERB/IMRB=`$60/$60`、
  TCDCR=`$50`、vector 68 table=`$00FC03EA`，Timer D running flag、machine scheduler與
  deadline均已清除；Timer C繼續運作。
- 有界續跑的下一個gate在289,332 instructions、234 interrupts、2,979,596 clocks：
  `$FC6B38` 對UCR `$FFFA29`重寫同值`$88`。Hatari對應trace也為`$88→$88`；這是
  後續USART重設／設定切片，不併入本規格。
- 完整目前240,000筆CPU corpus、固定ROM、全測試、vet與CLI build通過。

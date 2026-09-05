# 096 — ST MFP ACIA interrupt channel enable

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定EmuTOS在IKBD與MIDI ACIA初始化後，將MFP interrupt channel 6加入
既有Timer C channel bit5。後續channel 4、Timer D重新設定、實際ACIA IRQ仲裁與CPU
vector不在範圍。

- **已確認（MC68901／Atari ST平台契約）**：IERB／IPRB／ISRB／IMRB分別位於
  `$FFFA09/$FFFA0D/$FFFA11/$FFFA15`；channel 6對應bit6，enable與mask是兩層控制。
- **強證據（固定Hatari 2.4.1外部oracle）**：固定ROM在VBL10/HBL198依序執行
  IMRB=`$20`、IERB=`$20`、IPRB=`$BF`、ISRB=`$BF`、IERB=`$60`、IMRB=`$60`；
  PC依序為 `$FC619A/$FC61A4/$FC61A8/$FC61AC/$FC61EA/$FC61F4`。整段沒有pending
  ACIA interrupt被送入CPU。
- **已確認（固定Talos正常路徑）**：規格094–095後在136,135 instructions、8 interrupts、
  1,578,342 clocks抵達IERB同值 `$20` write，D2=`6`、現有IERB/IMRB=`$20/$20`。

## typed行為與驗收

1. MIDI ACIA configured、IERB/IMRB=`$20`時，接受IERB同值 `$20`作序列起點。
2. 只依序接受IPRB=`$BF`、ISRB=`$BF`；兩者仍用write-zero-to-clear，並推進stage。
3. clear完成後才接受IERB `$20→$60`；IMRB在無pending時依既有契約寫 `$60`。
4. 跳步、錯值、重複與pending非零均失敗即關閉且原子不變。此切片不產生IRQ。
5. synthetic stage測試、固定ROM、完整corpus、全測試、vet與build通過，並抵達下一個
   typed gate後才升 **CONFORMED**。

## 驗收收據

- synthetic測試確認五段stage與禁止跳過clear直接寫`$60`。
- 固定ROM完成IERB/IMRB=`$60/$60`，在136,182 instructions、8 interrupts、
  1,578,882 clocks抵達下一個IERB同值`$60` gate；D0=`$FFFFFFEF`與Hatari trace確認
  下一切片為channel 4／Timer D重新設定。
- 完整corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。

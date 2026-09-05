# 095 — ST IKBD ACIA stale RDR read

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定EmuTOS在讀取並清除IKBD ACIA RDRF後，由另一條系統路徑再讀一次
data register，取得保留的RDR=`$F1`。一般empty-register行為、無限重讀、overrun、
新RX byte與IRQ routing不在範圍。

- **已確認（MC6850平台契約）**：讀data register會清RDRF；RDR是獨立資料latch，清除
  status不等於把其資料位元歸零。軟體仍應以RDRF判斷資料是否為新資料。
- **強證據（固定Hatari 2.4.1外部oracle）**：EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。VBL10/HBL59
  `$FC48B4`讀RDR=`$F1`並把status清成 `$02`；HBL198 `$FC06CE`再次讀 `$FFFC02`，
  Hatari仍回 `$F1`，沒有新的RDRF事件。
- **已確認（固定Talos正常路徑）**：MIDI ACIA控制初始化後，在136,113 instructions、
  8 interrupts、1,578,092 clocks由 `$FC06CE`讀 `$FFFC02` 時抵達typed fault。

## typed行為與驗收

1. 第一次有效讀RDR=`$F1`時清bit7／bit0並建立恰好一次stale-latch read allowance。
2. configured、RDRF=`0`且allowance存在時，下一次supervisor byte read回 `$F1`，status
   維持 `$02`，並消耗allowance；不得冒充新RX byte或重新assert IRQ。
3. allowance前的empty read、消耗後重讀、user或word access維持失敗即關閉。
4. synthetic測試、固定ROM、完整corpus、全測試、vet與build通過，並抵達下一個typed
   gate後才升 **CONFORMED**。

## 驗收收據

- synthetic測試確認唯一一次stale read回 `$F1`且不改status，allowance耗盡後重讀失敗。
- 固定ROM在136,113 instructions／1,578,092 clocks由 `$FC06CE`完成stale read，隨後
  繼續MFP channel 6 enable路徑。
- 完整corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。

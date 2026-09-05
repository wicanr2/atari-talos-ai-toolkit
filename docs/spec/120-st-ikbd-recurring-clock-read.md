# 120 — ST IKBD可重入讀時鐘週期

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理固定EmuTOS完成set-clock readback與首輪`flopvbl()`後，再次呼叫
`igetregs()`所產生的IKBD `$1C`請求。既有前兩輪收據保留；第三輪起改用可重入的
請求／回應週期，不為每一次輪詢新增硬編碼特例。日期時間仍採固定profile，不接host
wall-clock；自然完成本輪後的下一個裝置gate不在此規格猜補。

## 證據

- **已確認（固定EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/clock.c:976 igetregs`每次皆送單一
  `$1C`並等待`clockvec()`收齊`$FC + 6-byte`；檔案SHA-256
  `32bafcf2d8c2f404eca720c8541e54806d15d4ff0243202fa201bfbb64995794`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1、EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；95-VBL
  `acia,ikbd_all,mfp_exception,mfp_read,mfp_write,video_vbl` trace SHA-256
  `9c0f18cff88762f66e775b5241be37d342e6f3f545a8202810c6664ccb63b641`。
  VBL77 PC `$FC5154`寫`$FFFC02=$1C`，firmware記
  `IKBD_Cmd_ReadClock: 24 03 17 00 00 00`，證實回應仍為
  `$FC,$24,$03,$17,$00,$00,$00`。
- **已確認（固定Talos入口）**：1,085,703 instructions／548 interrupts／
  13,927,048 clocks，PC `$FC515A`，D2=`$1C`，bus fault address `$FFFC02`。

## typed行為

1. 僅在readback、首輪`flopvbl()`、ACIA與既有回應週期皆完成，且TDR／shift register
   可接受新frame時，接受supervisor byte `$1C`；錯值、重疊請求、user／錯寬度失敗即關閉。
2. 沿用MC6850 10個1,024-clock serial ticks提交frame receipt；每個完成請求使單調
   request count加一，由machine排程該輪回應，不以一次性boolean阻止後續輪詢。
3. 每輪沿用16-tick首byte與10-tick後續byte期限，回應固定
   `$FC,$24,$03,$17,$00,$00,$00`；RDR backpressure、MFP channel 6、GPIP4與vector `$46`
   契約不變。每輪保存獨立delivery clocks、payload與guest讀取順序；已遷移至timed bus的
   讀取另存read clock，尚未遷移的指令不得把instruction epoch冒充bus phase。七筆讀完才
   增加complete count。
4. 下一輪只能在前一輪七筆均被guest讀取後開始；開始時清目前輪次陣列，不改寫規格116／
   118的歷史收據。cold reset清全部recurring counters與當輪收據。

## 驗收

- synthetic測試覆蓋連續兩輪請求、10-tick完成、response期限、RDR backpressure、每輪
  receipt重置、單調counter、錯序與cold reset。
- 固定ROM自然完成VBL77請求及七筆回應，鎖定完整CPU／clock邊界並定位下一gate。
- 固定ROM、完整240,000筆CPU corpus、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## CONFORMED收據

- 第三輪`$1C` frame於clock 13,937,502完成；回應delivery clocks為
  `13,953,886 / 13,964,126 / 13,974,366 / 13,984,606 / 13,994,846 /
  14,005,086 / 14,015,326`，payload為`[FC,24,03,17,00,00,00]`。
- guest在1,092,926 instructions／558 interrupts／14,015,626 clocks收齊七筆；
  request／response completion counters皆為1，response scheduler已清空。該路徑的
  read指令尚未遷移至timed byte bus，因此不宣稱精確read bus phase。
- 後續正常執行至1,120,640 instructions／568 interrupts／14,318,580 clocks，下一gate
  是`$FF8800=$0E`。固定Hatari trace SHA-256
  `e8008be66d87fa348dd802202dee8d83e7c23bbe4909bb871efed0b41ca8ff91`顯示VBL90
  `flopvbl()`依序select R14、讀`$23`、寫回`$23`、讀FDC status `$E4`、再恢復`$23`；
  這是後續週期性媒體輪詢規格。
- synthetic連續兩輪、固定ROM、完整240,000筆CPU corpus、全測試、
  `go vet -stdmethods=false ./...`與`go build ./...`均通過。

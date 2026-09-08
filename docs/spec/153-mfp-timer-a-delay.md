# 153 — MFP Timer A 延遲模式與中斷 **CONFORMED**

先經 READY 建立契約再實作；以下限定範圍已通過驗收，不涵蓋完整遊戲或音訊時序。

## 證據與範圍

已確認：Motorola MC68901 手冊 §6.1.1、§6.2.1–2、§4.3，
<https://www.nxp.com/docs/en/reference-manual/MC68901UM.pdf>；固定 PDF 雜湊沿用規格 069。
Timer A 控制 1–7 分別除以 4、10、16、50、64、100、200；0 停止。
計數 0 表示 256；倒數穿過 1 時重載、輸出翻轉，啟用時產生 channel 13 中斷。
停止保存主計數、丟棄 prescaler 餘數；停止時 data 寫入同時載入主計數，
執行中只改下次 reload。較高 channel 優先，software EOI 阻擋同級與低級。

已確認的玩家阻塞：DM12EN NOCP 10-sector 私人盤，SHA
`a1e6c4e3573babf633afe51dc321eb6279e915137897ccef3b9c7f274cf04873`，
EmuTOS 1.3 ROM SHA 沿用規格 099；step 20344161、clocks 307257568，
ROM CPU 空間 PC `$FC6266` 向 `$FFFA19` 寫入延遲模式 1 被拒絕。

## 實作契約

- 使用既有 ST 8021248 Hz／MFP 2457600 Hz 比率（簡約為 15667/4800）。
  倒數以整數餘數累積，不逐次截斷週期。timed bus 在 access 前推進，
  machine boundary 也推進；STOP 選取下一個 Timer A timeout，不能直接跳過它。
  STOP 的 CPU 喚醒對齊既有偶數 bus clock，計數器仍保留有理數餘數。
- 起始相位定在控制寫入的 access clock；異步晶振相位與 reload 同時寫入競態
  不模擬，標為 **hardware-spec approximation**，不宣稱音訊／逐週期精確。
- TACR 支援 0–7 與 bit 4 輸出 reset；unused 高位忽略且讀零。
  執行中相同 mode 可重入且不重置 phase；改成另一非零 prescaler 拒絕，
  因手冊明定首個 timeout 不定。event-count／pulse-width 8–15 仍拒絕。
- TADR byte read 回目前主計數；word 存取與 user I/O 存取仍拒絕。
- timeout 只在 IERA bit 5 啟用時設 IPRA bit 5；masked 時仍可 pending。
  IERA 新增 Timer A bit，其餘尚未支援的新通道仍拒絕。IMRA Timer A mask 可於
  pending 時修改，不能清 pending。disable 依規格 151 清 pending、不清 ISR。
- A13 優先於 B6/B5/B4；任何 A-bank ISR 阻擋 B-bank。CPU 接受中斷才清 pending，
  software EOI 時設 ISRA bit 5，vector=`(VR & 0xf0) | 13`。
- reset 清 Timer A phase、counter、output 與診斷 timeout 次數。不改遊戲規則或存檔。

## 驗收

內部：七分頻、0=256、分段推進不漂移、start/stop/restart、active data 延後重載、
同值重入、unsupported 原子拒絕、IER/IMR/ISR、A/B 優先序與 CPU STOP 喚醒。
外部：同盤自 reset 重跑，通過舊 gate 並觀察 Timer A timeout，記錄中斷是否啟用及
下一個真實阻塞；用 Hatari 微型測試驗證核心契約。不複製 GPL 實作。

## 驗收收據

- `TestTimerA*` 通過；CPU 中斷與 STOP 喚醒由合成程式驗證。
- `TestTimerAOracleProbe` 自製程式於 Talos 與 Hatari 2.4.1 均得
  `[3,3,5,5,32,0]`。設定 `TALOS_TIMERA_ORACLE_OUT` 可匯出程式及 Hatari
  偵錯腳本；以 `timera-init.ini` 等待 VBL 10 再執行，避免尚未初始化 CPU。
  設定 `TALOS_TIMERA_HATARI_RECEIPT` 回讀外部收據驗證。
- 完整 Go 測試含外部 68000 語料、vet、建置及六項 EmuTOS Timer C／D 回歸通過。
- 私人 DM12EN 自 reset 五千萬步達 ENTER／RESUME；timeouts=11579728、
  IACK=0。遊戲此路徑沒有 A 中斷處理證據，不能用 timeout 數冒充。
  下一阻塞為 ENTER 輸入時 IKBD 非相對模式，見磁片驗證紀錄。

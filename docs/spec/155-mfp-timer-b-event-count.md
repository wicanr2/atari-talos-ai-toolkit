# 155 — Timer B 顯示事件計數

狀態：**CONFORMED**（先完成 READY 契約再實作；範圍限下述正常 50 Hz 模式）。

已確認：Motorola MC68901UM（實際文件 MC68HC901）§6.1.3、6.2 與 §4：
https://www.nxp.com/docs/en/reference-manual/MC68901UM.pdf
mode 8 計算 TBI 有效邊緣；AER bit 3=0 為下降緣。0 表示 256，穿過 1
重載並翻轉輸出；IERA bit 0 控制 channel 8，遮罩不丟 pending，軟體 EOI
阻擋同級與低級。執行中資料寫入只改下次重載，停止保存主計數。

已確認（Hatari 2.4.1 黑箱、自製 CPU 探針）：50 Hz 正常顯示每幀在
line 63..262、line clock 400 計數一次，共 200 次；兩幀 trace 共 400 次。
探針 `TestTimerBExportOracleProbe` 匯出至私人目錄；同 EmuTOS ROM，VBL 10
啟動。不是遊戲注入或 GPL 原始碼移植。日誌 `timerb-oracle-cpu.log`。
探針 SHA-256：`358436cce50abccbf10eadb0c5e6e56f3423a1e489d5e06b78fe922cdc05569a`；
trace SHA-256：`687e944df0d9170ff7cb82f73a902a32d040bd443bed1e32f1ba2dae9adce6ee`。
`TALOS_TIMERB_HATARI_TRACE` 回讀該 trace，逐筆核對 400 個邊緣及排程延遲修正。

契約：本切片只支援停止與 mode 8、bit 4 輸出 reset；其餘模式仍拒絕。
沿用已建模 50 Hz VBL（line 0 clock 64）與 512 clocks/line；從 frame origin
排 200 個下降緣，邊框不計數。支援同值重入、STOP 喚醒、A13 > B8 > B-bank。
STOP 只在主計數真正到期時喚醒，不能因中途一條掃描線讓仍停止的 CPU 執行指令。
非 50 Hz、非彩色標準模式及 AER 非零仍拒絕，不猜補 overscan／切頻／上升緣。
已排事件在 timed bus 與指令邊界前推進，reset 清診斷與相位。
這是正常畫面事件契約；未建模完整 raster palette 擷取，不宣稱逐像素相同。

驗收：200 次/幀、空白期不計數、停止/恢復、0=256、延後重載、遮罩及
EOI／優先序、未支援值原子拒絕、CPU STOP；原片正常開門、移動與招募另記。
原片輸入雜湊沿用磁片驗證紀錄，不改遊戲規則、存檔或原版資料。

## 收據

- 內部 Timer B 測試、Hatari 400 事件收據回讀、完整 Go 外部 CPU 語料、vet、建置通過。
- STOP 測試使用主計數 139，確認不是第一條掃描線就試圖執行停止中的 CPU。
  0=256 的到期測試亦涵蓋跨幀空白期。
- 同片正常路徑完成前進／轉向、ELIJA 候選面板與 RESURRECT。
  最終 Timer B events=237661、IACK=2305；不是只接受控制暫存器寫入。
- `docs/dm12en-elija-actions.json` 保存正常輸入及三個 framebuffer 檢查點。
  圖像與原片只存私人證據目錄；雜湊回歸證明可重現，不外推全遊戲規則。

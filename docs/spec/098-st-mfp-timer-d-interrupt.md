# 098 — ST MFP Timer D 週期與向量中斷

狀態：**CONFORMED**。

## 範圍與證據

本切片把規格 097 已啟動的 Timer D 接到週期排程、channel 4 pending、MFP level 6
仲裁、向量承認與 EmuTOS handler。Timer C 與其他 MFP channel 仍不在範圍。

- **已確認（MC68901 一手規格）**：Timer D control 2 是 delay mode prescaler ÷10，
  data `$00` 表示 256，因此每期 2,560 MFP clocks；Timer D 是 channel 4。interrupt
  pending register 保存請求，mask register 決定 IRQ 輸出；software EOI mode 在承認時
  設定 in-service bit，軟體以 ISRB write-zero-to-clear 結束服務。向量由 VR 高四位與
  channel 編號組成。
- **已確認（Motorola／NXP 一手規格）**：外部裝置可在 interrupt acknowledge cycle
  提供 8-bit vector；CPU 仍以請求 level 更新 SR interrupt mask，並用該 vector 的
  `vector*4` 位置取得 handler。既有 44-clock interrupt exception 契約維持不變；
  本切片不宣稱逐 pin IACK 波形一致。
- **強證據（固定 Hatari 2.4.1／EmuTOS 1.3 UK oracle）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。TCDCR `$52`
  寫入發生於 Hatari global clock 1,599,776，記錄 `data=256 ctrl=2 timer_cyc=2560`；
  第一次 timeout 在下一個可接受 instruction boundary 進 level 6 exception。Hatari 先記錄
  autovector 30 的表位址 `$78`，MFP IACK 再改為 vector 68、表位址 `$110`、handler
  `$FC7884`。handler 於 `$FC788A` 寫 ISRB=`$EF` 清 channel 4，Timer D 同時 reload。
- **已確認（固定 Talos 正常路徑）**：規格 097 在 136,210 instructions、8 interrupts、
  1,579,228 clocks 完成 TCDCR=`$52`，VR=`$48`、IERB/IMRB=`$70`。

## typed 行為

1. 成功的 timed TCDCR `$50→$52` write 以該 bus access clock 為 Timer D phase anchor。
   目前固定 EmuTOS 所走的 `MOVE.B` effective-address path 尚未提供 timed access，故正常
   machine path 暫以該指令完成 boundary 為 anchor，標為 **hardware-spec approximation**；
   待該 opcode bus phase 遷移後自動優先採 access clock，不把近似冒稱逐 cycle parity。
   每期使用整數有理數 `2560*8021248/2457600` CPU clocks，deadline 取累積值的 floor；
   不把每期各自 floor 後重複相加，以免長期漂移。
2. 每逢 deadline 自動 reload，設定 IPRB bit 4；已 pending 時保持該 bit，不建立無界事件。
3. IERB、IMRB 與 IPRB bit 4 同時為 1 時提出 level 6。若 CPU mask 阻擋，pending 保留；
   可接受時由 MFP 回 vector `(VR & $F0) | 4`。固定 VR=`$48` 因此為 68。
4. 接受成功後清 IPRB bit 4；VR bit 3 為 1 的 software EOI mode 同時設 ISRB bit 4。
   handler 寫 `$EF` 後依既有 write-zero-to-clear 契約清除。
5. CPU vectored interrupt API 只接受 level 1–6；遮罩、saved PC、frame、SR、STOP 喚醒與
   44 clocks 均沿用規格 074，差別只有 vector 由呼叫端提供。原 autovector API 保持相容。
6. direct register write 本身不建立 machine deadline；只有 machine 正常執行路徑會消費
   stage 8 transition 並啟動排程，避免測試 helper 偽造時間證據。

## 驗收與停止線

- synthetic 測試覆蓋有理數 recurrence、pending 保留、mask 拒絕、vector 68、software
  EOI 的 pending→in-service→clear，以及 CPU vectored frame／prefetch／44 clocks。
- 固定 ROM 必須從規格 097 邊界自然進入 `$FC7884`，由 guest 自己在 `$FC788A`
  清 ISRB；不得直接改 PC、IPRB 或 ISRB。
- 完整 CPU corpus、全測試、`go vet` 與 build 通過後才升 **CONFORMED**。

## 驗收收據

- 固定 Talos 正常路徑以 1,579,228 clocks 的 instruction boundary 作明示近似 anchor；
  第一個 deadline 為 1,587,583，CPU 在下一個 instruction boundary 接受請求。
- 137,138 instructions、9 interrupts、1,587,632 clocks 自然進入 `$FC7884`；入口
  SR mask=6、prefetch=`$52B9,$0000`、IPRB bit 4=0、ISRB bit 4=1，guest 隨後由
  `$FC788A` 寫 `$EF` 清除 in-service，未由 host 直接改狀態。
- 完整目前 240,000 筆 CPU corpus、固定 ROM、全測試、`go vet -stdmethods=false ./...`
  與 CLI build 通過。

# 083 — ST MFP Timer C delay-mode start

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 開機路徑把 Timer C 從停止狀態啟動為
delay mode、control=`5`（prescaler ÷64）的 TCDCR `$FFFA1D` 寫入。countdown、
counter capture、reload、timeout、IPRB、IRQ、IACK 與 Timer D 都不在本切片；
這些行為未接線前不得把本規格冒稱為 200 Hz system timer 已完成。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §6.2.2。Timer C 使用 TCDCR bits 6–4；control `5` 是 delay mode 的 ÷64；
  prescaler pulse 遞減 main counter，`$01` 後的下一 pulse reload TDR 並產生 timeout。
- **已確認（固定 EmuTOS 1.3 原始碼與 ROM）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/mfp.c:208`
  以 `xbtimer(2, 0x50, 192, int_timerc)` 初始化 Timer C。`setup_timer()` 先保留
  Timer D、停止 Timer C，再把 `$C0` 寫 TCDR，最後將 `$50` OR 進 TCDCR。
  ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  `$FC62AA: MOVE.B D1,$1D(A0)` 是啟動寫入。
- **強證據（固定 Hatari 2.4.1 原始碼與外部 oracle）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`。`src/mfp.c` 的 TCDCR handler
  只替換 Timer C bits 6–4並保留 Timer D bits 2–0；`MFPDiv[5]=64`，data 192
  得 12,288 MFP ticks。固定 trace 在 global FrameCycles 124,038 寫 `$50`，
  `mfp start CD handler=6 data=192 ctrl=5 timer_cyc=12288 first=true`。Hatari 的
  ST CPU／MFP clock 是 8,021,248／2,457,600 Hz，向下取整週期為 40,106 CPU clocks。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 TCDCR 與本切片的 start transition 清零。
2. TCDCR 由 `$00` 寫 `$50` 時，保留低三位 Timer D control、將 Timer C control
   設為 `5`，並 latch 一次 Timer C start transition。既有 TCDR／main counter
   必須是 `$C0`；其他資料值在 countdown 尚未接線前失敗即關閉。
3. `$00→$00` 沿用規格 069。規格 086 後續證實 `$50→$50` 是共用 helper 在
   保留 active Timer C 時停止 Timer D 的必要同值 write，因此不再列為錯誤；停止
   active Timer C、其他非零 Timer C control與未由規格 086涵蓋的 Timer D control
   仍回 `unsupported_device_state`，且 register 與 transition 原子不變。
4. byte read 回 TCDCR；supervisor data byte access 固定增加 4 wait clocks；user、
   word 與相鄰未映射位址契約不變。
5. start transition 是交給下一層 scheduler 的 typed 邊界，不等同 timeout 已排程。
   在後續 Timer C countdown／reload 規格接線前，正常路徑若需要觀察第一個 timeout
   必須停止；規格 084 已分配給中途遇到的 MC68000 ROL，不能當作 timer backlink。

## 驗收與停止線

- synthetic test涵蓋唯一允許的 `$00→$50`、Timer D bits 保留限制、錯誤資料、
  重複／停止／其他 control 原子失敗、reset、alias、wait、保護與寬度。
- 固定 EmuTOS 必須完成 `$FC62AA` 的 16-clock MOVE，讀回 TCDCR=`$50`。實測完成
  68,103 條、3 interrupts、963,104 clocks，TCDCR／TCDR／main=`$50/$C0/$C0`；
  完整 CPU state 抵達 `$FC6192` 的未實作 memory `ROL.W` opcode `$E378`
  （pipeline PC `$FC6196`、prefetch `$E378,$1238`）。
- 完整 232,500 筆 CPU corpus、固定 ROM、全測試、vet 與 build 通過後才升
  **CONFORMED**；此切片的完整驗收已通過。

## 玩家路徑、存檔與權利邊界

本切片只延伸公開 EmuTOS 的可重現開機路徑，不改 Dungeon Master 規則、畫面或
存檔。專案只保存公開來源定位、雜湊與導出的 typed 契約，不收錄 ROM、NXP PDF，
也不複製或連結 Hatari GPL 實作。

# 099 — ST MFP Timer C 週期與向量中斷

狀態：**CONFORMED**。

## 範圍與證據

本切片把規格 083／085 已啟動並 enable 的 Timer C 接到週期排程、channel 5 pending、
MFP B-bank 優先序仲裁與 EmuTOS 200 Hz handler。只支援目前已建模的 Timer C channel 5
與 Timer D channel 4；其他 MFP sources、counter capture 與逐 pin IACK 不在範圍。

- **已確認（MC68901 一手規格）**：Timer C control 5 是 delay mode prescaler ÷64，
  data `$C0` 是 192，因此每期 12,288 MFP clocks。Timer C 是 channel 5；數字較大的
  channel 優先。pending、mask、software EOI 與 vector 組成沿用規格 098。
- **已確認（固定 EmuTOS 1.3 source／ROM）**：`bios/mfp.c:208` 以
  `xbtimer(2, 0x50, 192, int_timerc)` 設定 Timer C，ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
- **強證據（固定 Hatari 2.4.1 oracle）**：TCDCR `$50` trace 記錄
  `handler=6 data=192 ctrl=5 timer_cyc=12288`。每次 timeout 由 level 6 exception
  進 IACK，將暫定 autovector 30 的 `$78` 改為 vector 69、表位址 `$114`、handler
  `$FC04DE`；handler 在 `$FC050A` 寫 ISRB=`$DF`。連續 trace 顯示自動 reload。
- **已確認（既有 Talos）**：Timer C 在 68,103 instructions、963,104 clocks 完成
  TCDCR=`$50`；IERB／IMRB bit 5 隨後啟用。尚無 scheduler，故舊收據在該點之後漏掉
  200 Hz interrupts，必須由本切片正常重跑取代。

## typed 行為

1. Timer C 每期使用累積有理數 `12288*8021248/2457600` CPU clocks，deadline 取
   累積 floor，避免每期個別截斷造成漂移。timed TCDCR start write 優先提供 anchor；
   固定 `MOVE.B` path 未遷移前，以完成該指令的 machine boundary 作明示的
   **hardware-spec approximation**。
2. timeout 自動 reload 並設定 IPRB bit 5；既有 pending 保持，不建立無界事件。
3. B-bank 只在 `IPRB & IERB & IMRB` 中選已建模的 channel 5／4，較高的 channel 5
   優先。software-EOI 模式下，候選不得越過相同或更高的 ISRB in-service channel。
4. CPU 接受 level 6 後，MFP 清候選 pending，software EOI 時設定同一 ISRB bit，向量為
   `(VR & $F0) | channel`。固定 Timer C 為 vector 69；guest 寫 `$DF` 清服務狀態。
5. direct register test 不啟動 machine scheduler；reset 清 phase、period 與 deadline。

## 驗收與停止線

- synthetic 測試涵蓋 Timer C 有理數週期、bit 5 pending、C 高於 D、in-service 阻擋、
  vector 69 與 pending→in-service→guest clear。
- 固定 ROM 從既有正常入口自然進 `$FC04DE`，不得直接改 PC／IPRB／ISRB；重跑並訂正
  Timer C start 之後受影響的里程碑收據。
- 完整 CPU corpus、固定 ROM、全測試、vet 與 build 通過後才升 **CONFORMED**。

## 驗收收據

- 固定 Talos timed start phase=962,844；第一個累積 floor deadline=1,002,950，CPU 在
  下一個 instruction boundary 接受。72,342 instructions、5 interrupts、1,003,004 clocks
  自然進 `$FC04DE`；入口 prefetch=`$52B8,$04BA`、IPRB bit 5=0、ISRB bit 5=1，
  guest 後續由 `$FC050A` 寫 `$DF` 清除。
- 接入後，IKBD `$F1` 讀取的新正常收據是 128,378 instructions、21 interrupts、
  1,509,022 clocks；Timer D system start 是 136,285／23／1,581,256；Timer D handler
  是 137,213／24／1,589,660。舊的漏 Timer C 計數已由本規格訂正。
- 完整目前 240,000 筆 CPU corpus、固定 ROM、全測試、`go vet -stdmethods=false ./...`
  與 CLI build 均通過。
